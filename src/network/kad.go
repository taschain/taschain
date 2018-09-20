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
	alpha           = 3  // 并发限制
	bucketSize      = 35 // kad桶大小
	maxReplacements = 10 // kad 预备桶成员大小
	maxSetupCheckCount = 12

	hashBits          = 256
	nBuckets          = hashBits / 15       // kad桶数量
	bucketMinDistance = hashBits - nBuckets // 最近桶的对数距离

	refreshInterval    = 5 * time.Minute
	checkInterval	= 	12 * time.Second
	copyNodesInterval  = 30 * time.Second
	nodeBondExpiration = 5 * time.Second
	seedMinTableTime   = 5 * time.Minute
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
	seeds []*Node           // 启动节点列表
	rand    *mrand.Rand       // 随机数生成器

	refreshReq chan chan struct{}
	initDone   chan struct{}
	closeReq   chan struct{}
	closed     chan struct{}

	net  NetInterface
	self *Node

	setupCheckCount int
}

type bondProc struct {
	err  error
	n    *Node
	done chan struct{}
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

func newKad(t NetInterface, ourID NodeID, ourAddr *nnet.UDPAddr,  seeds []*Node) (*Kad, error) {
	kad := &Kad{
		net:        t,
		self:       newNode(ourID, ourAddr.IP, ourAddr.Port),
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
		kad.buckets[i] = &bucket{
		}
	}
	kad.seedRand()
	kad.loadSeedNodes(false)
	go kad.loop()
	return kad, nil
}

//print 打印桶成员信息
func (kad *Kad) print() {
	for i, b := range kad.buckets {
		for _, n := range b.entries {
			Logger.Debugf(" [kad] bucket:%v id:%v  addr: IP:%v    Port:%v...", i,n.Id.GetHexString(),n.Ip, n.Port)
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
		cpy.sha = makeSha256Hash(n.Id[:])
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
	if len(cl.entries) > 0 && cl.entries[0].Id == targetID {
		return cl.entries[0]
	}

	return nil
}


func (kad *Kad) resolve(targetID NodeID) *Node {
	hash := makeSha256Hash(targetID[:])
	kad.mutex.Lock()
	cl := kad.closest(hash, 1)
	kad.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].Id == targetID {
		return cl.entries[0]
	}
	// 找不到，开始向临近节点询问
	result := kad.Lookup(targetID)
	for _, n := range result {
		if n.Id == targetID {
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

	asked[kad.self.Id] = true

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
			if !asked[n.Id] {
				asked[n.Id] = true
				pendingQueries++
				go func() {
					r, err := kad.net.findNode(n.Id, n.addr(), targetID)
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
			if n != nil && !seen[n.Id] {
				seen[n.Id] = true
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
		refresh        = time.NewTicker(refreshInterval)
		check        = time.NewTicker(checkInterval)
		copyNodes      = time.NewTicker(copyNodesInterval)
		refreshDone    = make(chan struct{})           // where doRefresh reports completion
		waiting        = []chan struct{}{kad.initDone} // holds waiting callers while doRefresh runs
	)
	defer refresh.Stop()
	defer copyNodes.Stop()

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
			if kad.setupCheckCount >  maxSetupCheckCount {
				check.Stop()
			} else {
				kad.setupCheckCount = kad.setupCheckCount +1
				go kad.doCheck()
			}

		case <-refreshDone:
			for _, ch := range waiting {
				close(ch)
			}
			waiting, refreshDone = nil, nil
		case <-copyNodes.C:
			go kad.copyBondedNodes()
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
	kad.print()
	kad.loadSeedNodes(true)

	kad.lookup(kad.self.Id, false)

	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		kad.lookup(target, false)
	}
}


func (kad *Kad) doCheck() {

	Logger.Debugf("doCheck ... bucket size:%v ", kad.len())
	//if kad.len() <= len(kad.nursery) * 3{
	kad.refresh()
	///}
}


func (kad *Kad) loadSeedNodes(bond bool) {
	seeds := make([]*Node, 0, 16)

	seeds = append(seeds, kad.seeds...)
	if bond {
		seeds = kad.pingAll(seeds)
	}
	for i := range seeds {
		seed := seeds[i]
		kad.add(seed)
	}
}


func (kad *Kad) copyBondedNodes() {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	now := time.Now()
	for _, b := range kad.buckets {
		for _, n := range b.entries {
			if now.Sub(n.addedAt) >= seedMinTableTime {
				//kad.db.updateNode(n)
			}
		}
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
			nn, _ := kad.pingNode(n.Id, n.addr())
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

func (kad *Kad) pingNode( id NodeID, addr *nnet.UDPAddr) (*Node, error) {

	if id == kad.self.Id {
		return nil, errors.New("is self")
	}

	var node *Node
	node = kad.find(id)
	age := nodeBondExpiration
	fails := 0
	pinged :=  false
	if node != nil {
		age = time.Since(node.pingAt)
		fails = int(node.fails)
		node.pingAt = time.Now()
		pinged = node.pinged
	}

	var result error
	if !pinged && ( fails > 0 || age >= nodeBondExpiration) {
		kad.net.ping(id, addr)
	}

	return node, result
}

func (kad *Kad) onPingNode( id NodeID, addr *nnet.UDPAddr) (*Node, error) {

	if id == kad.self.Id {
		return nil, errors.New("is self")
	}

	var node *Node
	node = kad.find(id)
	if node == nil {
		node = newNode(id, addr.IP, addr.Port)
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
		if n.Id == kad.self.Id {
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

func (kad *Kad) addIP(b *bucket, ip nnet.IP) bool {

	return true
}

func (kad *Kad) removeIP(b *bucket, ip nnet.IP) {

}

func (kad *Kad) addReplacement(b *bucket, n *Node) {
	for _, e := range b.replacements {
		if e.Id == n.Id {
			return
		}
	}
	if !kad.addIP(b, n.Ip) {
		return
	}
	var removed *Node
	b.replacements, removed = pushNode(b.replacements, n, maxReplacements)
	if removed != nil {
		kad.removeIP(b, removed.Ip)
	}
}

func (kad *Kad) replace(b *bucket, last *Node) *Node {
	if len(b.entries) == 0 || b.entries[len(b.entries)-1].Id != last.Id {
		return nil
	}
	if len(b.replacements) == 0 {
		kad.deleteInBucket(b, last)
		return nil
	}
	r := b.replacements[kad.rand.Intn(len(b.replacements))]
	b.replacements = deleteNode(b.replacements, r)
	b.entries[len(b.entries)-1] = r
	kad.removeIP(b, last.Ip)
	return r
}

func (b *bucket) bump(n *Node) bool {
	for i := range b.entries {
		if b.entries[i].Id == n.Id {
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
	if len(b.entries) >= bucketSize || !kad.addIP(b, n.Ip) {
		return false
	}
	b.entries, _ = pushNode(b.entries, n, bucketSize)
	b.replacements = deleteNode(b.replacements, n)
	n.addedAt = time.Now()

	return true
}

func (kad *Kad) deleteInBucket(b *bucket, n *Node) {
	b.entries = deleteNode(b.entries, n)
	kad.removeIP(b, n.Ip)
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
		if list[i].Id == n.Id {
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
