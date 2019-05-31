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

package network

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	mrand "math/rand"
	nnet "net"
	"sort"
	"sync"
	"time"
)

const (
	alpha              = 3  // 并发限制
	bucketSize         = 35 // kad桶大小
	maxReplacements    = 10 // kad 预备桶成员大小
	maxSetupCheckCount = 12

	hashBits          = 256
	nBuckets          = hashBits / 15       // kad桶数量
	bucketMinDistance = hashBits - nBuckets // 最近桶的对数距离

	refreshInterval    = 5 * time.Minute
	checkInterval      = 12 * time.Second
	nodeBondExpiration = 5 * time.Second
)

// getSha256Hash 计算哈希
func makeSha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

//Kad kad
type Kad struct {
	mutex   sync.Mutex        // 保护成员 buckets, bucket content, nursery, rand
	buckets [nBuckets]*bucket // 根据节点距离排序的节点的索引
	seeds   []*Node           // 启动节点列表
	rand    *mrand.Rand       // 随机数生成器

	refreshReq chan chan struct{}
	initDone   chan struct{}
	closeReq   chan struct{}
	closed     chan struct{}

	net  NetInterface
	self *Node

	setupCheckCount int
}

type NetInterface interface {
	ping(NodeID, *nnet.UDPAddr) error
	findNode(toid NodeID, addr *nnet.UDPAddr, target NodeID) ([]*Node, error)
	close()
}

type bucket struct {
	entries      []*Node // 活动节点
	replacements []*Node // 备用补充节点
}

func newKad(t NetInterface, ourID NodeID, ourAddr *nnet.UDPAddr, seeds []*Node) (*Kad, error) {
	kad := &Kad{
		net:        t,
		self:       NewNode(ourID, ourAddr.IP, ourAddr.Port),
		refreshReq: make(chan chan struct{}),
		initDone:   make(chan struct{}),
		closeReq:   make(chan struct{}),
		closed:     make(chan struct{}),
		rand:       mrand.New(mrand.NewSource(0)),
	}
	if err := kad.setFallbackNodes(seeds); err != nil {
		return nil, err
	}
	for i := range kad.buckets {
		kad.buckets[i] = &bucket{}
	}
	kad.seedRand()
	kad.loadSeedNodes(false)
	go kad.loop()
	return kad, nil
}

//print 打印桶成员信息
func (kad *Kad) print() {
	Logger.Debugf(" [kad] print bucket size: %v", kad.len())

	for i, b := range kad.buckets {
		for _, n := range b.entries {
			Logger.Debugf(" [kad] print bucket:%v id:%v  addr: IP:%v    Port:%v", i, n.ID.GetHexString(), n.IP, n.Port)
		}
	}

	return
}

func (kad *Kad) seedRand() {
	var b [8]byte
	crand.Read(b[:])

	kad.mutex.Lock()
	kad.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	kad.mutex.Unlock()
}

func (kad *Kad) Self() *Node {
	return kad.self
}

func (kad *Kad) readRandomNodes(buf []*Node) (n int) {
	if !kad.isInitDone() {
		return 0
	}
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	var buckets [][]*Node
	for _, b := range kad.buckets {
		if len(b.entries) > 0 {
			buckets = append(buckets, b.entries[:])
		}
	}
	if len(buckets) == 0 {
		return 0
	}
	for i := len(buckets) - 1; i > 0; i-- {
		j := kad.rand.Intn(len(buckets))
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}
	var i, j int
	for ; i < len(buf); i, j = i+1, (j+1)%len(buckets) {
		b := buckets[j]
		buf[i] = &(*b[0])
		buckets[j] = b[1:]
		if len(b) == 1 {
			buckets = append(buckets[:j], buckets[j+1:]...)
		}
		if len(buckets) == 0 {
			break
		}
	}
	return i + 1
}

func (kad *Kad) Close() {
	select {
	case <-kad.closed:
	case kad.closeReq <- struct{}{}:
		<-kad.closed
	}
}

func (kad *Kad) setFallbackNodes(nodes []*Node) error {
	for _, n := range nodes {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap node %v (%v)", n, err)
		}
	}
	kad.seeds = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		cpy := *n
		cpy.sha = makeSha256Hash(n.ID[:])
		kad.seeds = append(kad.seeds, &cpy)
	}
	return nil
}

func (kad *Kad) isInitDone() bool {
	select {
	case <-kad.initDone:
		return true
	default:
		return false
	}
}

//Find 只在桶里查找
func (kad *Kad) find(targetID NodeID) *Node {
	hash := makeSha256Hash(targetID[:])
	kad.mutex.Lock()
	cl := kad.closest(hash, 1)
	kad.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].ID == targetID {
		return cl.entries[0]
	}

	return nil
}

func (kad *Kad) resolve(targetID NodeID) *Node {
	hash := makeSha256Hash(targetID[:])
	kad.mutex.Lock()
	cl := kad.closest(hash, 1)
	kad.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].ID == targetID {
		return cl.entries[0]
	}
	// 找不到，开始向临近节点询问
	result := kad.Lookup(targetID)
	for _, n := range result {
		if n.ID == targetID {
			return n
		}
	}
	return nil
}

func (kad *Kad) Lookup(targetID NodeID) []*Node {
	return kad.lookup(targetID, true)
}

func (kad *Kad) lookup(targetID NodeID, refreshIfEmpty bool) []*Node {
	var (
		target         = makeSha256Hash(targetID[:])
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		pendingQueries = 0
		result         *nodesByDistance
	)

	asked[kad.self.ID] = true

	for {
		kad.mutex.Lock()
		result = kad.closest(target, bucketSize)
		kad.mutex.Unlock()
		if len(result.entries) > 0 || !refreshIfEmpty {
			break
		}
		<-kad.refresh()
		refreshIfEmpty = false
	}

	for {
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				pendingQueries++
				go func() {
					r, err := kad.net.findNode(n.ID, n.addr(), targetID)
					if err != nil {
					}
					reply <- kad.pingAll(r)
				}()
			}
		}
		if pendingQueries == 0 {
			break
		}
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, bucketSize)
			}
		}
		pendingQueries--
	}
	return result.entries
}

func (kad *Kad) refresh() <-chan struct{} {
	done := make(chan struct{})
	select {
	case kad.refreshReq <- done:
	case <-kad.closed:
		close(done)
	}
	return done
}

func (kad *Kad) loop() {
	var (
		refresh     = time.NewTicker(refreshInterval)
		check       = time.NewTicker(checkInterval)
		refreshDone = make(chan struct{})           // where doRefresh reports completion
		waiting     = []chan struct{}{kad.initDone} // holds waiting callers while doRefresh runs
	)
	defer refresh.Stop()

	go kad.doRefresh(refreshDone)

loop:
	for {
		select {
		case <-refresh.C:
			kad.seedRand()
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go kad.doRefresh(refreshDone)
			}
		case req := <-kad.refreshReq:
			waiting = append(waiting, req)
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go kad.doRefresh(refreshDone)
			}
		case <-check.C:
			kad.setupCheckCount = kad.setupCheckCount + 1
			go kad.doCheck()

		case <-refreshDone:
			for _, ch := range waiting {
				close(ch)
			}
			waiting, refreshDone = nil, nil
		case <-kad.closeReq:
			break loop
		}
	}

	if kad.net != nil {
		kad.net.close()
	}
	if refreshDone != nil {
		<-refreshDone
	}
	for _, ch := range waiting {
		close(ch)
	}
	close(kad.closed)
}

func (kad *Kad) doRefresh(done chan struct{}) {
	defer close(done)
	kad.loadSeedNodes(true)

	kad.lookup(kad.self.ID, false)

	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		kad.lookup(target, false)
	}
}

func (kad *Kad) doCheck() {

	Logger.Debugf("[kad] check ... bucket size:%v ", kad.len())
	if kad.len() <= len(kad.seeds) || kad.setupCheckCount < maxSetupCheckCount {
		kad.refresh()
	}
}

func (kad *Kad) loadSeedNodes(bond bool) {

	if bond {
		kad.pingAll(kad.seeds)
	}

	for i := range kad.seeds {
		kad.add(kad.seeds[i])
	}
}

func (kad *Kad) closest(target []byte, nresults int) *nodesByDistance {

	close := &nodesByDistance{target: target}
	for _, b := range kad.buckets {
		for _, n := range b.entries {
			close.push(n, nresults)
		}
	}
	return close
}

func (kad *Kad) len() (n int) {
	for _, b := range kad.buckets {
		n += len(b.entries)
	}
	return n
}

func (kad *Kad) pingAll(nodes []*Node) (result []*Node) {

	rc := make(chan *Node, len(nodes))
	for i := range nodes {
		go func(n *Node) {
			nn, _ := kad.pingNode(n.ID, n.addr())
			rc <- nn
		}(nodes[i])
	}
	for range nodes {
		if n := <-rc; n != nil {
			result = append(result, n)
		}
	}
	return result
}

func (kad *Kad) pingNode(id NodeID, addr *nnet.UDPAddr) (*Node, error) {

	if id == kad.self.ID {
		return nil, errors.New("is self")
	}

	var node *Node
	node = kad.find(id)
	age := nodeBondExpiration
	fails := 0
	pinged := false
	if node != nil {
		age = time.Since(node.pingAt)
		fails = int(node.fails)
		node.pingAt = time.Now()
		pinged = node.pinged
	}

	var result error
	if !pinged && (fails > 0 || age >= nodeBondExpiration) {
		kad.net.ping(id, addr)
	}

	return node, result
}

func (kad *Kad) onPingNode(id NodeID, addr *nnet.UDPAddr) (*Node, error) {

	if id == kad.self.ID {
		return nil, errors.New("is self")
	}

	var node *Node
	node = kad.find(id)
	if node == nil {
		node = NewNode(id, addr.IP, addr.Port)
		kad.add(node)

	}
	node.pinged = true
	return node, nil
}

func (kad *Kad) hasPinged(id NodeID) bool {
	node := kad.find(id)

	if node != nil {
		return node.pinged
	}
	return false
}

func (kad *Kad) bucket(sha []byte) *bucket {
	d := logDistance(kad.self.sha, sha)
	if d <= bucketMinDistance {
		return kad.buckets[0]
	}
	return kad.buckets[d-bucketMinDistance-1]
}

func (kad *Kad) add(new *Node) {

	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	b := kad.bucket(new.sha)
	if !kad.bumpOrAdd(b, new) {
		kad.addReplacement(b, new)
	}
}

func (kad *Kad) stuff(nodes []*Node) {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	for _, n := range nodes {
		if n.ID == kad.self.ID {
			continue // don't add self
		}
		b := kad.bucket(n.sha)
		if len(b.entries) < bucketSize {
			kad.bumpOrAdd(b, n)
		}
	}
}

func (kad *Kad) delete(node *Node) {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	kad.deleteInBucket(kad.bucket(node.sha), node)
}

func (kad *Kad) addReplacement(b *bucket, n *Node) {
	for _, e := range b.replacements {
		if e.ID == n.ID {
			return
		}
	}

	b.replacements, _ = pushNode(b.replacements, n, maxReplacements)

}

func (kad *Kad) replace(b *bucket, last *Node) *Node {
	if len(b.entries) == 0 || b.entries[len(b.entries)-1].ID != last.ID {
		return nil
	}
	if len(b.replacements) == 0 {
		kad.deleteInBucket(b, last)
		return nil
	}
	r := b.replacements[kad.rand.Intn(len(b.replacements))]
	b.replacements = deleteNode(b.replacements, r)
	b.entries[len(b.entries)-1] = r

	return r
}

func (b *bucket) bump(n *Node) bool {
	for i := range b.entries {
		if b.entries[i].ID == n.ID {
			// move it to the front
			copy(b.entries[1:], b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}

func (kad *Kad) bumpOrAdd(b *bucket, n *Node) bool {
	if b.bump(n) {
		return true
	}
	if len(b.entries) >= bucketSize {
		return false
	}
	b.entries, _ = pushNode(b.entries, n, bucketSize)
	b.replacements = deleteNode(b.replacements, n)
	n.addedAt = time.Now()

	return true
}

func (kad *Kad) deleteInBucket(b *bucket, n *Node) {
	b.entries = deleteNode(b.entries, n)
}

func pushNode(list []*Node, n *Node, max int) ([]*Node, *Node) {
	if len(list) < max {
		list = append(list, nil)
	}
	removed := list[len(list)-1]
	copy(list[1:], list)
	list[0] = n
	return list, removed
}

func deleteNode(list []*Node, n *Node) []*Node {
	for i := range list {
		if list[i].ID == n.ID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

type nodesByDistance struct {
	entries []*Node
	target  []byte
}

func (h *nodesByDistance) push(n *Node, maxElems int) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distanceCompare(h.target, h.entries[i].sha, n.sha) > 0
	})
	if len(h.entries) < maxElems {
		h.entries = append(h.entries, n)
	}
	if ix == len(h.entries) {
	} else {
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}
