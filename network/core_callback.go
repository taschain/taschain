package network

import "C"

//export OnP2PRecved
func OnP2PRecved(id uint64, session uint32, data []byte) {
	netCore.OnRecved(id, session, data)
}

//export OnP2PChecked
func OnP2PChecked(p2pType uint32, privateIP string, publicIP string) {
	netCore.OnChecked(p2pType, privateIP, publicIP)
}

//export OnP2PListened
func OnP2PListened(ip string, port uint16, latency uint64) {
}

//export OnP2PAccepted
func OnP2PAccepted(id uint64, session uint32, p2pType uint32) {
	netCore.OnAccepted(id, session, p2pType)
}

//export OnP2PConnected
func OnP2PConnected(id uint64, session uint32, p2pType uint32) {
	netCore.OnConnected(id, session, p2pType)
}

//export OnP2PDisconnected
func OnP2PDisconnected(id uint64, session uint32, p2pCode uint32) {
	netCore.OnDisconnected(id, session, p2pCode)
}

//export OnP2PSendWaited
func OnP2PSendWaited(session uint32, peerID uint64) {
	netCore.OnSendWaited(peerID, session)
}
