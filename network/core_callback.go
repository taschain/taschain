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

import "C"

//export OnP2PRecved
func OnP2PRecved(id uint64, session uint32, data []byte) {
	netCore.OnRecved(id, session, data)
}

//export OnP2PChecked
func OnP2PChecked(p2p_type uint32, private_ip string, public_ip string) {
	netCore.OnChecked(p2p_type, private_ip, public_ip)
}

//export OnP2PListened
func OnP2PListened(ip string, port uint16, latency uint64) {
}

//export OnP2PAccepted
func OnP2PAccepted(id uint64, session uint32, p2p_type uint32) {
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

//export OnP2PSendWaited
func OnP2PSendWaited(session uint32, peer_id uint64) {
	netCore.OnSendWaited(peer_id, session)
}
