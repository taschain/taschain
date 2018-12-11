//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"common"
)

var (
	ErrClientQuit                = errors.New("client is closed")
	ErrNoResult                  = errors.New("no result in JSON-RPC response")
	ErrSubscriptionQueueOverflow = errors.New("subscription queue overflow")
)

const (
	// Timeouts
	tcpKeepAliveInterval = 30 * time.Second
	defaultDialTimeout   = 10 * time.Second // used when dialing if the context has no deadline
	defaultWriteTimeout  = 10 * time.Second // used for calls if the context has no deadline
	subscribeTimeout     = 5 * time.Second  // overall timeout eth_subscribe, rpc_modules calls
)

const (
	maxClientSubscriptionBuffer = 8000
)

type BatchElem struct {
	Method string
	Args   []interface{}

	Result interface{}

	Error error
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (msg *jsonrpcMessage) isNotification() bool {
	return msg.ID == nil && msg.Method != ""
}

func (msg *jsonrpcMessage) isResponse() bool {
	return msg.hasValidID() && msg.Method == "" && len(msg.Params) == 0
}

func (msg *jsonrpcMessage) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *jsonrpcMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

type Client struct {
	idCounter   uint32
	connectFunc func(ctx context.Context) (net.Conn, error)
	isHTTP      bool

	writeConn net.Conn

	// for dispatch
	close       chan struct{}
	didQuit     chan struct{}                  // closed when client quits
	reconnected chan net.Conn                  // where write/reconnect sends the new connection
	readErr     chan error                     // errors from read
	readResp    chan []*jsonrpcMessage         // valid messages from read
	requestOp   chan *requestOp                // for registering response IDs
	sendDone    chan error                     // signals write completion, releases write lock
	respWait    map[string]*requestOp          // active requests
	subs        map[string]*ClientSubscription // active subscriptions
}

type requestOp struct {
	ids  []json.RawMessage
	err  error
	resp chan *jsonrpcMessage
	sub  *ClientSubscription
}

func (op *requestOp) wait(ctx context.Context) (*jsonrpcMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		return DialHTTP(rawurl)
	case "ws", "wss":
		return DialWebsocket(ctx, rawurl, "")
	case "":
		return DialIPC(ctx, rawurl)
	default:
		return nil, fmt.Errorf("no known transport for URL scheme %q", u.Scheme)
	}
}

func newClient(initctx context.Context, connectFunc func(context.Context) (net.Conn, error)) (*Client, error) {
	conn, err := connectFunc(initctx)
	if err != nil {
		return nil, err
	}
	_, isHTTP := conn.(*httpConn)

	c := &Client{
		writeConn:   conn,
		isHTTP:      isHTTP,
		connectFunc: connectFunc,
		close:       make(chan struct{}),
		didQuit:     make(chan struct{}),
		reconnected: make(chan net.Conn),
		readErr:     make(chan error),
		readResp:    make(chan []*jsonrpcMessage),
		requestOp:   make(chan *requestOp),
		sendDone:    make(chan error, 1),
		respWait:    make(map[string]*requestOp),
		subs:        make(map[string]*ClientSubscription),
	}
	if !isHTTP {
		go c.dispatch(conn)
	}
	return c, nil
}

func (c *Client) nextID() json.RawMessage {
	id := atomic.AddUint32(&c.idCounter, 1)
	return []byte(strconv.FormatUint(uint64(id), 10))
}

func (c *Client) SupportedModules() (map[string]string, error) {
	var result map[string]string
	ctx, cancel := context.WithTimeout(context.Background(), subscribeTimeout)
	defer cancel()
	err := c.CallContext(ctx, &result, "rpc_modules")
	return result, err
}

func (c *Client) Close() {
	if c.isHTTP {
		return
	}
	select {
	case c.close <- struct{}{}:
		<-c.didQuit
	case <-c.didQuit:
	}
}

func (c *Client) Call(result interface{}, method string, args ...interface{}) error {
	ctx := context.Background()
	return c.CallContext(ctx, result, method, args...)
}

func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	msg, err := c.newMessage(method, args...)
	if err != nil {
		return err
	}
	op := &requestOp{ids: []json.RawMessage{msg.ID}, resp: make(chan *jsonrpcMessage, 1)}

	if c.isHTTP {
		err = c.sendHTTP(ctx, op, msg)
	} else {
		err = c.send(ctx, op, msg)
	}
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel it when it quits.
	switch resp, err := op.wait(ctx); {
	case err != nil:
		return err
	case resp.Error != nil:
		return resp.Error
	case len(resp.Result) == 0:
		return ErrNoResult
	default:
		return json.Unmarshal(resp.Result, &result)
	}
}

func (c *Client) BatchCall(b []BatchElem) error {
	ctx := context.Background()
	return c.BatchCallContext(ctx, b)
}

func (c *Client) BatchCallContext(ctx context.Context, b []BatchElem) error {
	msgs := make([]*jsonrpcMessage, len(b))
	op := &requestOp{
		ids:  make([]json.RawMessage, len(b)),
		resp: make(chan *jsonrpcMessage, len(b)),
	}
	for i, elem := range b {
		msg, err := c.newMessage(elem.Method, elem.Args...)
		if err != nil {
			return err
		}
		msgs[i] = msg
		op.ids[i] = msg.ID
	}

	var err error
	if c.isHTTP {
		err = c.sendBatchHTTP(ctx, op, msgs)
	} else {
		err = c.send(ctx, op, msgs)
	}

	// Wait for all responses to come back.
	for n := 0; n < len(b) && err == nil; n++ {
		var resp *jsonrpcMessage
		resp, err = op.wait(ctx)
		if err != nil {
			break
		}
		// Find the element corresponding to this response.
		// The element is guaranteed to be present because dispatch
		// only sends valid IDs to our channel.
		var elem *BatchElem
		for i := range msgs {
			if bytes.Equal(msgs[i].ID, resp.ID) {
				elem = &b[i]
				break
			}
		}
		if resp.Error != nil {
			elem.Error = resp.Error
			continue
		}
		if len(resp.Result) == 0 {
			elem.Error = ErrNoResult
			continue
		}
		elem.Error = json.Unmarshal(resp.Result, elem.Result)
	}
	return err
}

// EthSubscribe registers a subscripion under the "eth" namespace.
func (c *Client) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	return c.Subscribe(ctx, "eth", channel, args...)
}

// ShhSubscribe registers a subscripion under the "shh" namespace.
func (c *Client) ShhSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	return c.Subscribe(ctx, "shh", channel, args...)
}

func (c *Client) Subscribe(ctx context.Context, namespace string, channel interface{}, args ...interface{}) (*ClientSubscription, error) {
	// Check type of channel first.
	chanVal := reflect.ValueOf(channel)
	if chanVal.Kind() != reflect.Chan || chanVal.Type().ChanDir()&reflect.SendDir == 0 {
		panic("first argument to Subscribe must be a writable channel")
	}
	if chanVal.IsNil() {
		panic("channel given to Subscribe must not be nil")
	}
	if c.isHTTP {
		return nil, ErrNotificationsUnsupported
	}

	msg, err := c.newMessage(namespace+subscribeMethodSuffix, args...)
	if err != nil {
		return nil, err
	}
	op := &requestOp{
		ids:  []json.RawMessage{msg.ID},
		resp: make(chan *jsonrpcMessage),
		sub:  newClientSubscription(c, namespace, chanVal),
	}

	if err := c.send(ctx, op, msg); err != nil {
		return nil, err
	}
	if _, err := op.wait(ctx); err != nil {
		return nil, err
	}
	return op.sub, nil
}

func (c *Client) newMessage(method string, paramsIn ...interface{}) (*jsonrpcMessage, error) {
	params, err := json.Marshal(paramsIn)
	if err != nil {
		return nil, err
	}
	return &jsonrpcMessage{Version: "2.0", ID: c.nextID(), Method: method, Params: params}, nil
}

func (c *Client) send(ctx context.Context, op *requestOp, msg interface{}) error {
	select {
	case c.requestOp <- op:
		err := c.write(ctx, msg)
		c.sendDone <- err
		return err
	case <-ctx.Done():
		// This can happen if the client is overloaded or unable to keep up with
		// subscription notifications.
		return ctx.Err()
	case <-c.didQuit:
		return ErrClientQuit
	}
}

func (c *Client) write(ctx context.Context, msg interface{}) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	// The previous write failed. Try to establish a new connection.
	if c.writeConn == nil {
		if err := c.reconnect(ctx); err != nil {
			return err
		}
	}
	c.writeConn.SetWriteDeadline(deadline)
	err := json.NewEncoder(c.writeConn).Encode(msg)
	if err != nil {
		c.writeConn = nil
	}
	return err
}

func (c *Client) reconnect(ctx context.Context) error {
	newconn, err := c.connectFunc(ctx)
	if err != nil {
		common.DefaultLogger.Debugf(fmt.Sprintf("reconnect failed: %v", err))
		return err
	}
	select {
	case c.reconnected <- newconn:
		c.writeConn = newconn
		return nil
	case <-c.didQuit:
		newconn.Close()
		return ErrClientQuit
	}
}

func (c *Client) dispatch(conn net.Conn) {
	// Spawn the initial read loop.
	go c.read(conn)

	var (
		lastOp        *requestOp    // tracks last send operation
		requestOpLock = c.requestOp // nil while the send lock is held
		reading       = true        // if true, a read loop is running
	)
	defer close(c.didQuit)
	defer func() {
		c.closeRequestOps(ErrClientQuit)
		conn.Close()
		if reading {
			// Empty read channels until read is dead.
			for {
				select {
				case <-c.readResp:
				case <-c.readErr:
					return
				}
			}
		}
	}()

	for {
		select {
		case <-c.close:
			return

			// Read path.
		case batch := <-c.readResp:
			for _, msg := range batch {
				switch {
				case msg.isNotification():
					c.handleNotification(msg)
				case msg.isResponse():
					c.handleResponse(msg)
				default:
					// TODO: maybe close
				}
			}

		case err := <-c.readErr:
			common.DefaultLogger.Debug(fmt.Sprintf("<-readErr: %v", err))
			c.closeRequestOps(err)
			conn.Close()
			reading = false

		case newconn := <-c.reconnected:
			common.DefaultLogger.Debug(fmt.Sprintf("<-reconnected: (reading=%t) %v", reading, conn.RemoteAddr()))
			if reading {
				// Wait for the previous read loop to exit. This is a rare case.
				conn.Close()
				<-c.readErr
			}
			go c.read(newconn)
			reading = true
			conn = newconn

			// Send path.
		case op := <-requestOpLock:
			// Stop listening for further send ops until the current one is done.
			requestOpLock = nil
			lastOp = op
			for _, id := range op.ids {
				c.respWait[string(id)] = op
			}

		case err := <-c.sendDone:
			if err != nil {
				for _, id := range lastOp.ids {
					delete(c.respWait, string(id))
				}
			}

			requestOpLock = c.requestOp
			lastOp = nil
		}
	}
}

func (c *Client) closeRequestOps(err error) {
	didClose := make(map[*requestOp]bool)

	for id, op := range c.respWait {

		delete(c.respWait, id)

		if !didClose[op] {
			op.err = err
			close(op.resp)
			didClose[op] = true
		}
	}
	for id, sub := range c.subs {
		delete(c.subs, id)
		sub.quitWithError(err, false)
	}
}

func (c *Client) handleNotification(msg *jsonrpcMessage) {
	if !strings.HasSuffix(msg.Method, notificationMethodSuffix) {
		common.DefaultLogger.Debug(fmt.Sprint("dropping non-subscription message: ", msg))
		return
	}
	var subResult struct {
		ID     string          `json:"subscription"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(msg.Params, &subResult); err != nil {
		common.DefaultLogger.Debug(fmt.Sprint("dropping invalid subscription message: ", msg))
		return
	}
	if c.subs[subResult.ID] != nil {
		c.subs[subResult.ID].deliver(subResult.Result)
	}
}

func (c *Client) handleResponse(msg *jsonrpcMessage) {
	op := c.respWait[string(msg.ID)]
	if op == nil {
		common.DefaultLogger.Debug(fmt.Sprintf("unsolicited response %v", msg))
		return
	}
	delete(c.respWait, string(msg.ID))

	if op.sub == nil {
		op.resp <- msg
		return
	}

	defer close(op.resp)
	if msg.Error != nil {
		op.err = msg.Error
		return
	}
	if op.err = json.Unmarshal(msg.Result, &op.sub.subid); op.err == nil {
		go op.sub.start()
		c.subs[op.sub.subid] = op.sub
	}
}

// Reading happens on a dedicated goroutine.

func (c *Client) read(conn net.Conn) error {
	var (
		buf json.RawMessage
		dec = json.NewDecoder(conn)
	)
	readMessage := func() (rs []*jsonrpcMessage, err error) {
		buf = buf[:0]
		if err = dec.Decode(&buf); err != nil {
			return nil, err
		}
		if isBatch(buf) {
			err = json.Unmarshal(buf, &rs)
		} else {
			rs = make([]*jsonrpcMessage, 1)
			err = json.Unmarshal(buf, &rs[0])
		}
		return rs, err
	}

	for {
		resp, err := readMessage()
		if err != nil {
			c.readErr <- err
			return err
		}
		c.readResp <- resp
	}
}

type ClientSubscription struct {
	client    *Client
	etype     reflect.Type
	channel   reflect.Value
	namespace string
	subid     string
	in        chan json.RawMessage

	quitOnce sync.Once     // ensures quit is closed once
	quit     chan struct{} // quit is closed when the subscription exits
	errOnce  sync.Once     // ensures err is closed once
	err      chan error
}

func newClientSubscription(c *Client, namespace string, channel reflect.Value) *ClientSubscription {
	sub := &ClientSubscription{
		client:    c,
		namespace: namespace,
		etype:     channel.Type().Elem(),
		channel:   channel,
		quit:      make(chan struct{}),
		err:       make(chan error, 1),
		in:        make(chan json.RawMessage),
	}
	return sub
}

func (sub *ClientSubscription) Err() <-chan error {
	return sub.err
}

func (sub *ClientSubscription) Unsubscribe() {
	sub.quitWithError(nil, true)
	sub.errOnce.Do(func() { close(sub.err) })
}

func (sub *ClientSubscription) quitWithError(err error, unsubscribeServer bool) {
	sub.quitOnce.Do(func() {

		close(sub.quit)
		if unsubscribeServer {
			sub.requestUnsubscribe()
		}
		if err != nil {
			if err == ErrClientQuit {
				err = nil // Adhere to subscription semantics.
			}
			sub.err <- err
		}
	})
}

func (sub *ClientSubscription) deliver(result json.RawMessage) (ok bool) {
	select {
	case sub.in <- result:
		return true
	case <-sub.quit:
		return false
	}
}

func (sub *ClientSubscription) start() {
	sub.quitWithError(sub.forward())
}

func (sub *ClientSubscription) forward() (err error, unsubscribeServer bool) {
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.quit)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.in)},
		{Dir: reflect.SelectSend, Chan: sub.channel},
	}
	buffer := list.New()
	defer buffer.Init()
	for {
		var chosen int
		var recv reflect.Value
		if buffer.Len() == 0 {
			// Idle, omit send case.
			chosen, recv, _ = reflect.Select(cases[:2])
		} else {
			// Non-empty buffer, send the first queued item.
			cases[2].Send = reflect.ValueOf(buffer.Front().Value)
			chosen, recv, _ = reflect.Select(cases)
		}

		switch chosen {
		case 0: // <-sub.quit
			return nil, false
		case 1: // <-sub.in
			val, err := sub.unmarshal(recv.Interface().(json.RawMessage))
			if err != nil {
				return err, true
			}
			if buffer.Len() == maxClientSubscriptionBuffer {
				return ErrSubscriptionQueueOverflow, true
			}
			buffer.PushBack(val)
		case 2:                             // sub.channel<-
			cases[2].Send = reflect.Value{} // Don't hold onto the value.
			buffer.Remove(buffer.Front())
		}
	}
}

func (sub *ClientSubscription) unmarshal(result json.RawMessage) (interface{}, error) {
	val := reflect.New(sub.etype)
	err := json.Unmarshal(result, val.Interface())
	return val.Elem().Interface(), err
}

func (sub *ClientSubscription) requestUnsubscribe() error {
	var result interface{}
	return sub.client.Call(&result, sub.namespace+unsubscribeMethodSuffix, sub.subid)
}
