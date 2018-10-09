package network
import "C"

//export OnP2PRecved
func OnP2PRecved(id uint64, session uint32, data []byte) {
	netCore.OnRecved(id, session, data)
}

//export OnP2PChecked
func OnP2PChecked(p2p_type uint32, private_ip string, public_ip string) {
	//fmt.Printf("%v %v %v %v\n", "OnP2PChecked", p2p_type, private_ip, public_ip)
}

//export OnP2PListened
func OnP2PListened(ip string, port uint16, latency uint64) {
	//fmt.Printf("%v %v %v %v\n", "OnP2PListened", ip, port, latency)
}

//export OnP2PAccepted
func OnP2PAccepted(id uint64, session uint32, p2p_type uint32) {
	//fmt.Printf("%v %v %v %v\n", "OnP2PAccepted", id, session, p2p_type)
	netCore.OnAccepted(id, session, p2p_type)
}

//export OnP2PConnected
func OnP2PConnected(id uint64, session uint32, p2p_type uint32) {
	netCore.OnConnected(id, session, p2p_type)
}

//export OnP2PDisconnected
func OnP2PDisconnected(id uint64, session uint32, p2p_code uint32) {
	netCore.OnDisconnected(id, session, p2p_code)
}
