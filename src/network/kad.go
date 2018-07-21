package network

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	alpha           = 3  // 并发限制
	bucketSize      = 16 // kad桶大小
	maxReplacements = 10 // kad 预备桶成员大小

	hashBits          = 256
	nBuckets          = hashBits / 15       // kad桶数量
	bucketMinDistance = hashBits - nBuckets // 最近桶的对数距离

	maxBondingPingPongs = 16 // 最大ping/pong数量限制
	maxFindnodeFailures = 5  // 节点最大失败数量

	refreshInterval    =30 * time.Second
	revalidateInterval = 30 * time.Minute
	checkInterval	= 	3 * time.Second
	copyNodesInterval  = 30 * time.Second
	nodeBondExpiration = 5 * time.Second
	seedMinTableTime   = 5 * time.Minute
	seedCount          = 30
	seedMaxAge         = 5 * 24 * time.Hour
)

// SHA256Hash 计算哈希
func SHA256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

//Kad kad
type Kad struct {
	mutex   sync.Mutex        // 保护成员 buckets, bucket content, nursery, rand
	buckets [nBuckets]*bucket // 根据节点距离排序的节点的索引
	nursery []*Node           // 启动节点列表
	rand    *mrand.Rand       // 随机数生成器

	//db *nodeDB //已知节点缓存数据库

	refreshReq chan chan struct{}
	initDone   chan struct{}
	closeReq   chan struct{}
	closed     chan struct{}

	bondmu    sync.Mutex
	bonding   map[NodeID]*bondproc
	bondslots chan struct{} // limits total number of active bonding processes

	nodeAddedHook func(*Node) // for testing

	net  transport
	self *Node // metadata of the local node
}

type bondproc struct {
	err  error
	n    *Node
	done chan struct{}
}

// transport 使用UDP实现通信.
// 这只是一个接口我们不用打开UDP套接字，生成私有key就能测试
type transport interface {
	ping(NodeID, *net.UDPAddr) error
	waitping(NodeID) error
	findnode(toid NodeID, addr *net.UDPAddr, target NodeID) ([]*Node, error)
	close()
	print()
}

// 用来储存发现的节点，节点1️以他们的最后活动时间排序
// 最后活跃的节点出现在最前面.
type bucket struct {
	entries      []*Node // 活动节点
	replacements []*Node // 备用补充节点
}

func newKad(t transport, ourID NodeID, ourAddr *net.UDPAddr, nodeDBPath string, bootnodes []*Node) (*Kad, error) {
	// If no node database was given, use an in-memory one
	//db, err := newNodeDB(nodeDBPath, Version, ourID)
	// if err != nil {
	// 	return nil, err
	// }
	kad := &Kad{
		net:        t,
		self:       NewNode(ourID, ourAddr.IP, ourAddr.Port),
		bonding:    make(map[NodeID]*bondproc),
		bondslots:  make(chan struct{}, maxBondingPingPongs),
		refreshReq: make(chan chan struct{}),
		initDone:   make(chan struct{}),
		closeReq:   make(chan struct{}),
		closed:     make(chan struct{}),
		rand:       mrand.New(mrand.NewSource(0)),
		// ips:        netutil.DistinctNetSet{Subnet: tableSubnet, Limit: tableIPLimit},
	}
	if err := kad.setFallbackNodes(bootnodes); err != nil {
		return nil, err
	}
	for i := 0; i < cap(kad.bondslots); i++ {
		kad.bondslots <- struct{}{}
	}
	for i := range kad.buckets {
		kad.buckets[i] = &bucket{
			// ips: netutil.DistinctNetSet{Subnet: bucketSubnet, Limit: bucketIPLimit},
		}
	}
	kad.seedRand()
	kad.loadSeedNodes(false)
	// Start the background expiration goroutine after loading seeds so that the search for
	// seed nodes also considers older nodes that would otherwise be removed by the
	// expiration.
	//kad.db.ensureExpirer()
	go kad.loop()
	return kad, nil
}

func (kad *Kad) seedRand() {
	var b [8]byte
	crand.Read(b[:])

	kad.mutex.Lock()
	kad.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	kad.mutex.Unlock()
}

// Self returns the local node.
// The returned node should not be modified by the caller.
func (kad *Kad) Self() *Node {
	return kad.self
}


// ReadRandomNodes fills the given slice with random nodes from the
// table. It will not write the same node more than once. The nodes in
// the slice are copies and can be modified by the caller.
func (kad *Kad) ReadRandomNodes(buf []*Node) (n int) {
	if !kad.isInitDone() {
		return 0
	}
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	// Find all non-empty buckets and get a fresh slice of their entries.
	var buckets [][]*Node
	for _, b := range kad.buckets {
		if len(b.entries) > 0 {
			buckets = append(buckets, b.entries[:])
		}
	}
	if len(buckets) == 0 {
		return 0
	}
	// Shuffle the buckets.
	for i := len(buckets) - 1; i > 0; i-- {
		j := kad.rand.Intn(len(buckets))
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}
	// Move head of each bucket into buf, removing buckets that become empty.
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

// Close terminates the network listener and flushes the node database.
func (kad *Kad) Close() {
	select {
	case <-kad.closed:
		// already closed.
	case kad.closeReq <- struct{}{}:
		<-kad.closed // wait for refreshLoop to end.
	}
}

// setFallbackNodes sets the initial points of contact. These nodes
// are used to connect to the network if the table is empty and there
// are no known nodes in the database.
func (kad *Kad) setFallbackNodes(nodes []*Node) error {
	for _, n := range nodes {
		if err := n.validateComplete(); err != nil {
			return fmt.Errorf("bad bootstrap/fallback node %v (%v)", n, err)
		}
	}
	kad.nursery = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		cpy := *n
		// Recompute cpy.sha because the node might not have been
		// created by NewNode or ParseNode.
		//cpy.sha = crypto.Keccak256Hash(n.Id[:])
		kad.nursery = append(kad.nursery, &cpy)
	}
	return nil
}

// isInitDone returns whether the table's initial seeding procedure has completed.
func (kad *Kad) isInitDone() bool {
	select {
	case <-kad.initDone:
		return true
	default:
		return false
	}
}

//Find 只在桶里查找
func (kad *Kad) Find(targetID NodeID) *Node {
	// If the node is present in the local table, no
	// network interaction is required.
	//hash := crypto.Keccak256Hash(targetID[:])
	hash := SHA256Hash(targetID[:])
	kad.mutex.Lock()
	cl := kad.closest(hash, 1)
	kad.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].Id == targetID {
		return cl.entries[0]
	}

	return nil
}

// Resolve searches for a specific node with the given ID.
// It returns nil if the node could not be found.
func (kad *Kad) Resolve(targetID NodeID) *Node {
	// If the node is present in the local table, no
	// network interaction is required.
	//hash := crypto.Keccak256Hash(targetID[:])
	hash := SHA256Hash(targetID[:])
	kad.mutex.Lock()
	cl := kad.closest(hash, 1)
	kad.mutex.Unlock()
	if len(cl.entries) > 0 && cl.entries[0].Id == targetID {
		return cl.entries[0]
	}
	// Otherwise, do a network lookup.
	result := kad.Lookup(targetID)
	for _, n := range result {
		if n.Id == targetID {
			return n
		}
	}
	return nil
}

// Lookup performs a network search for nodes close
// to the given target. It approaches the target by querying
// nodes that are closer to it on each iteration.
// The given target does not need to be an actual node
// identifier.
func (kad *Kad) Lookup(targetID NodeID) []*Node {
	return kad.lookup(targetID, true)
}

func (kad *Kad) lookup(targetID NodeID, refreshIfEmpty bool) []*Node {
	var (
		target         = SHA256Hash(targetID[:])
		asked          = make(map[NodeID]bool)
		seen           = make(map[NodeID]bool)
		reply          = make(chan []*Node, alpha)
		pendingQueries = 0
		result         *nodesByDistance
	)

	//fmt.Printf("lookup  id:%v \n", targetID)

	// don't query further if we hit ourself.
	// unlikely to happen often in practice.
	asked[kad.self.Id] = true

	for {
		kad.mutex.Lock()
		// generate initial result set
		result = kad.closest(target, bucketSize)
		kad.mutex.Unlock()
		if len(result.entries) > 0 || !refreshIfEmpty {
			break
		}
		// The result set is empty, all nodes were dropped, refresh.
		// We actually wait for the refresh to complete here. The very
		// first query will hit this case and run the bootstrapping
		// logic.
		<-kad.refresh()
		refreshIfEmpty = false
	}

	for {
		// ask the alpha closest nodes that we haven't asked yet
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.Id] {
				asked[n.Id] = true
				pendingQueries++
				go func() {
					// Find potential neighbors to bond with
					r, err := kad.net.findnode(n.Id, n.addr(), targetID)
					if err != nil {
						// Bump the failure counter to detect and evacuate non-bonded entries
						// fails := kad.db.findFails(n.Id) + 1
						// kad.db.updateFindFails(n.Id, fails)
						// log.Trace("Bumping findnode failure counter", "id", n.Id, "failcount", fails)

						// if fails >= maxFindnodeFailures {
						// 	log.Trace("Too many findnode failures, dropping", "id", n.Id, "failcount", fails)
						// 	kad.delete(n)
						// }
					}
					reply <- kad.bondall(r)
				}()
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
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

// loop schedules refresh, revalidate runs and coordinates shutdown.
func (kad *Kad) loop() {
	var (
		revalidate     = time.NewTimer(kad.nextRevalidateTime())
		refresh        = time.NewTicker(refreshInterval)
		check        = time.NewTicker(checkInterval)
		copyNodes      = time.NewTicker(copyNodesInterval)
		revalidateDone = make(chan struct{})
		refreshDone    = make(chan struct{})           // where doRefresh reports completion
		waiting        = []chan struct{}{kad.initDone} // holds waiting callers while doRefresh runs
	)
	defer refresh.Stop()
	defer revalidate.Stop()
	defer copyNodes.Stop()

	// Start initial refresh.
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
		case <-refreshDone:
			for _, ch := range waiting {
				close(ch)
			}
			waiting, refreshDone = nil, nil
		case <-revalidate.C:
			go kad.doRevalidate(revalidateDone)
		case <-check.C:
			go kad.doCheck()
		case <-revalidateDone:
			revalidate.Reset(kad.nextRevalidateTime())
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
	//	kad.db.close()
	close(kad.closed)
}

// doRefresh performs a lookup for a random target to keep buckets
// full. seed nodes are inserted if the table is empty (initial
// bootstrap or discarded faulty peers).
func (kad *Kad) doRefresh(done chan struct{}) {
	defer close(done)

	//fmt.Printf("doRefresh  \n")

	// Load nodes from the database and insert
	// them. This should yield a few previously seen nodes that are
	// (hopefully) still alive.
	kad.loadSeedNodes(true)

	// Run self lookup to discover new neighbor nodes.
	kad.lookup(kad.self.Id, false)

	// The Kademlia paper specifies that the bucket refresh should
	// perform a lookup in the least recently used bucket. We cannot
	// adhere to this because the findnode target is a 512bit value
	// (not hash-sized) and it is not easily possible to generate a
	// sha3 preimage that falls into a chosen bucket.
	// We perform a few lookups with a random target instead.
	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		kad.lookup(target, false)
	}
}

func (kad *Kad) loadSeedNodes(bond bool) {
	seeds := make([]*Node, 0, 16)
	//kad.db.querySeeds(seedCount, seedMaxAge)
	//fmt.Printf("loadSeedNodes...\n")

	seeds = append(seeds, kad.nursery...)
	if bond {
		seeds = kad.bondall(seeds)
	}
	for i := range seeds {
		seed := seeds[i]
		// age := log.Lazy{Fn: func() interface{} { return time.Since(kad.db.bondTime(seed.Id)) }}
		// log.Debug("Found seed node in database", "id", seed.Id, "addr", seed.addr(), "age", age)
		kad.add(seed)
	}
}


func (kad *Kad) doCheck() {
	//fmt.Printf("doCheck ... bucket size:%v \n", kad.len())
	//if kad.len() <= len(kad.nursery) * 3{
		kad.refresh()
	///}
	kad.net.print()

}

// doRevalidate checks that the last node in a random bucket is still live
// and replaces or deletes the node if it isn't.
func (kad *Kad) doRevalidate(done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	//fmt.Printf("doRevalidate ... bucket size:%v \n", kad.len())
	kad.Print()
	last, bi := kad.nodeToRevalidate()
	last =nil;
	//n := 12
	//
	//hello := ""
	//for i := 0; i < n; i++ {
	//	hello += "KAD"
	//}
	//kad.net.SendDataToAll([]byte(hello))
	if last == nil {
		// No non-empty bucket found.
		return
	}

	// Ping the selected node and wait for a pong.
	err := kad.ping(last.Id, last.addr())

	kad.mutex.Lock()
	defer kad.mutex.Unlock()
	b := kad.buckets[bi]
	if err == nil {
		// The node responded, move it to the front.
		//log.Debug("Revalidated node", "b", bi, "id", last.Id)
		b.bump(last)
		return
	}
	// No reply received, pick a replacement or delete the node if there aren't
	// any replacements.
	if r := kad.replace(b, last); r != nil {
		//log.Debug("Replaced dead node", "b", bi, "id", last.Id, "ip", last.Ip, "r", r.Id, "rip", r.Ip)
	} else {
		//log.Debug("Removed dead node", "b", bi, "id", last.Id, "ip", last.Ip)
	}
}

// nodeToRevalidate returns the last node in a random, non-empty bucket.
func (kad *Kad) nodeToRevalidate() (n *Node, bi int) {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	for _, bi = range kad.rand.Perm(len(kad.buckets)) {
		b := kad.buckets[bi]
		if len(b.entries) > 0 {
			last := b.entries[len(b.entries)-1]
			return last, bi
		}
	}
	return nil, 0
}

func (kad *Kad) nextRevalidateTime() time.Duration {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	return time.Duration(kad.rand.Int63n(int64(revalidateInterval)))
}

// copyBondedNodes adds nodes from the table to the database if they have been in the table
// longer then minTableTime.
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

// closest returns the n nodes in the table that are closest to the
// given id. The caller must hold kad.mutex.
func (kad *Kad) closest(target []byte, nresults int) *nodesByDistance {
	//fmt.Printf("kad closest ... bucket size:%v \n", kad.len())

	// This is a very wasteful way to find the closest nodes but
	// obviously correct. I believe that tree-based buckets would make
	// this easier to implement efficiently.
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

//Print 打印桶成员信息
func (kad *Kad) Print() {
	for _, b := range kad.buckets {
		for _, n := range b.entries {
			fmt.Printf("----- kad ---  addr: IP:%v    Port:%v...\n", n.Ip, n.Port)
		}
	}
	return
}

// bondall bonds with all given nodes concurrently and returns
// those nodes for which bonding has probably succeeded.
func (kad *Kad) bondall(nodes []*Node) (result []*Node) {

	//fmt.Printf("bondall   %v...\n", kad.len())

	rc := make(chan *Node, len(nodes))
	for i := range nodes {
		go func(n *Node) {
			nn, _ := kad.bond(false, n.Id, n.addr(), n.Port)
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

func (kad *Kad) bond(pinged bool, id NodeID, addr *net.UDPAddr, port int) (*Node, error) {

	if id == kad.self.Id {
		return nil, errors.New("is self")
	}
	if pinged && !kad.isInitDone() {

		return nil, errors.New("still initializing")
	}

	var node *Node
	node = kad.Find(id)
	age := nodeBondExpiration
	fails := 0
	bonded :=  false
	if node != nil {
		age = time.Since(node.bondAt)
		fails = int(node.fails)
		node.bondAt = time.Now()
		bonded = node.bonded
	}

	var result error
	if !bonded && ( fails > 0 || age >= nodeBondExpiration) {
		kad.bondmu.Lock()
		w := kad.bonding[id]
		if w != nil {
			kad.bondmu.Unlock()
			<-w.done
		} else {
			w = &bondproc{done: make(chan struct{})}
			kad.bonding[id] = w
			kad.bondmu.Unlock()
			// Do the ping/pong. The result goes into w.
			kad.pingpong(w, pinged, id, addr, port)
			// Unregister the process after it's done.
			kad.bondmu.Lock()
			delete(kad.bonding, id)
			kad.bondmu.Unlock()
		}
		// Retrieve the bonding results
		result = w.err
		if result == nil {
			node = w.n
		}
		if node != nil {
			node.bonded = true
			kad.add(node)
		}
	}

	return node, result
	//return nil, nil
}

func (kad *Kad) pingpong(w *bondproc, pinged bool, id NodeID, addr *net.UDPAddr, tcpPort int) {
	// Request a bonding slot to limit network usage
	<-kad.bondslots
	defer func() { kad.bondslots <- struct{}{} }()

	// Ping the remote side and wait for a pong.
	if w.err = kad.ping(id, addr); w.err != nil {
		close(w.done)
		return
	}
	if !pinged {
		kad.net.waitping(id)
	}
	w.n = NewNode(id, addr.IP, addr.Port)
	close(w.done)
}

// ping a remote endpoint and wait for a reply, also updating the node
// database accordingly.
func (kad *Kad) ping(id NodeID, addr *net.UDPAddr) error {
	//kad.db.updateLastPing(id, time.Now())
	//fmt.Printf("ping ...\n")
	if err := kad.net.ping(id, addr); err != nil {
		return err
	}
	//kad.db.updateBondTime(id, time.Now())
	return nil
}

func (kad *Kad) hasBond(id NodeID) bool {
	node := kad.Find(id)

	if node != nil {
		return node.bonded
	}
	return false
}
// bucket returns the bucket for the given node ID hash.
func (kad *Kad) bucket(sha []byte) *bucket {
	d := logdist(kad.self.sha, sha)
	if d <= bucketMinDistance {
		return kad.buckets[0]
	}
	return kad.buckets[d-bucketMinDistance-1]
}

// add attempts to add the given node its corresponding bucket. If the
// bucket has space available, adding the node succeeds immediately.
// Otherwise, the node is added if the least recently active node in
// the bucket does not respond to a ping packet.
//
// The caller must not hold kad.mutex.
func (kad *Kad) add(new *Node) {

	//fmt.Printf("Kad add node id:%v\n", new.Id.B58String())

	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	b := kad.bucket(new.sha)
	if !kad.bumpOrAdd(b, new) {
		// Node is not in table. Add it to the replacement list.
		kad.addReplacement(b, new)
	}
}

// stuff adds nodes the table to the end of their corresponding bucket
// if the bucket is not full. The caller must not hold kad.mutex.
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

// delete removes an entry from the node table (used to evacuate
// failed/non-bonded discovery peers).
func (kad *Kad) delete(node *Node) {
	kad.mutex.Lock()
	defer kad.mutex.Unlock()

	kad.deleteInBucket(kad.bucket(node.sha), node)
}

func (kad *Kad) addIP(b *bucket, ip net.IP) bool {
	// if netutil.IsLAN(ip) {
	// 	return true
	// }
	// if !kad.Ips.Add(ip) {
	// 	log.Debug("IP exceeds table limit", "ip", ip)
	// 	return false
	// }
	// if !b.Ips.Add(ip) {
	// 	log.Debug("IP exceeds bucket limit", "ip", ip)
	// 	kad.Ips.Remove(ip)
	// 	return false
	// }
	return true
}

func (kad *Kad) removeIP(b *bucket, ip net.IP) {
	// if netutil.IsLAN(ip) {
	// 	return
	// }
	// kad.Ips.Remove(ip)
	// b.Ips.Remove(ip)
}

func (kad *Kad) addReplacement(b *bucket, n *Node) {
	for _, e := range b.replacements {
		if e.Id == n.Id {
			return // already in list
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

// replace removes n from the replacement list and replaces 'last' with it if it is the
// last entry in the bucket. If 'last' isn't the last entry, it has either been replaced
// with someone else or became active.
func (kad *Kad) replace(b *bucket, last *Node) *Node {
	if len(b.entries) == 0 || b.entries[len(b.entries)-1].Id != last.Id {
		// Entry has moved, don't replace it.
		return nil
	}
	// Still the last entry.
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

// bump moves the given node to the front of the bucket entry list
// if it is contained in that list.
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

// bumpOrAdd moves n to the front of the bucket entry list or adds it if the list isn't
// full. The return value is true if n is in the bucket.
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
	if kad.nodeAddedHook != nil {
		kad.nodeAddedHook(n)
	}
	return true
}

func (kad *Kad) deleteInBucket(b *bucket, n *Node) {
	b.entries = deleteNode(b.entries, n)
	kad.removeIP(b, n.Ip)
}

// pushNode adds n to the front of list, keeping at most max items.
func pushNode(list []*Node, n *Node, max int) ([]*Node, *Node) {
	if len(list) < max {
		list = append(list, nil)
	}
	removed := list[len(list)-1]
	copy(list[1:], list)
	list[0] = n
	return list, removed
}

// deleteNode removes n from list.
func deleteNode(list []*Node, n *Node) []*Node {
	for i := range list {
		if list[i].Id == n.Id {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// nodesByDistance is a list of nodes, ordered by
// distance to target.
type nodesByDistance struct {
	entries []*Node
	target  []byte
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *Node, maxElems int) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return distcmp(h.target, h.entries[i].sha, n.sha) > 0
	})
	if len(h.entries) < maxElems {
		h.entries = append(h.entries, n)
	}
	if ix == len(h.entries) {
		// farther away than all nodes we already have.
		// if there was room for it, the node is now the last element.
	} else {
		// slide existing entries down to make room
		// this will overwrite the entry we just appended.
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}
