package network
import "C"

func OnP2PRecved(id uint64, session uint32, data []byte) {
	netCore.OnRecved(id, session, data)
}

func OnP2PChecked(p2p_type uint32, private_ip string, public_ip string) {
	netCore.OnChecked(p2p_type, private_ip, public_ip)
}

func OnP2PListened(ip string, port uint16, latency uint64) {
}

func OnP2PAccepted(id uint64, session uint32, p2p_type uint32) {
	netCore.OnAccepted(id, session, p2p_type)
}

func OnP2PConnected(id uint64, session uint32, p2p_type uint32) {
	netCore.OnConnected(id, session, p2p_type)
}

func OnP2PDisconnected(id uint64, session uint32, p2p_code uint32) {
	netCore.OnDisconnected(id, session, p2p_code)
}

func OnP2PSendWaited(session uint32,peer_id uint64) {
	netCore.OnSendWaited(peer_id, session)
}