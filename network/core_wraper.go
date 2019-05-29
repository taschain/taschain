package network

/*
#cgo LDFLAGS: -L ./ -lp2pcore -lstdc++

#include "p2p_api.h"

#include <stdint.h>
#include <string.h>

void OnP2PRecved();

void OnP2PListened();

void OnP2PChecked();

void OnP2PAccepted();

void OnP2PConnected();

void OnP2PDisconnected();

void OnP2PSendWaited();


void on_p2p_recved(uint64_t id, uint32_t session, char* data, uint32_t size)
{
	_GoBytes_ _data = {data, size, size};
	OnP2PRecved(id, session, _data);
}

void on_p2p_checked(uint32_t type, const char* private_ip, const char* public_ip)
{
    _GoString_ _private_ip = {private_ip, strlen(private_ip)};
    _GoString_ _public_ip = {public_ip, strlen(public_ip)};
    OnP2PChecked(type, _private_ip, _public_ip);
}

void on_p2p_listened(const char* ip, uint16_t port, uint64_t latency)
{
	_GoString_ _ip = {ip, strlen(ip)};
	OnP2PListened(_ip, port, latency);
}

void on_p2p_accepted(uint64_t id, uint32_t session, uint32_t type)
{
	OnP2PAccepted(id, session, type);
}

void on_p2p_connected(uint64_t id, uint32_t session, uint32_t type)
{
	OnP2PConnected(id, session, type);
}

void on_p2p_disconnected(uint64_t id, uint32_t session, uint32_t code)
{
	OnP2PDisconnected(id, session, code);
}

void on_p2p_send_waited(uint32_t session, uint64_t peer_id)
{
	OnP2PSendWaited(session, peer_id);
}

void wrap_p2p_config(uint64_t id)
{
	struct p2p_callback callback = { 0 };
	callback.recved = on_p2p_recved;
	callback.checked = on_p2p_checked;
	callback.listened = on_p2p_listened;
	callback.accepted = on_p2p_accepted;
	callback.connected = on_p2p_connected;
	callback.disconnected = on_p2p_disconnected;
	p2p_config(id, callback);

}

void wrap_p2p_send_callback()
{
	p2p_send_callback(on_p2p_send_waited);

}

*/
import "C"
import "unsafe"

func P2PConfig(id uint64) {
	C.wrap_p2p_config(C.uint64_t(id))
	C.wrap_p2p_send_callback()
}

func P2PProxy(ip string, port uint16) {
	C.p2p_proxy(C.CString(ip), C.ushort(port))
}

func P2PListen(ip string, port uint16) {
	C.p2p_listen(C.CString(ip), C.ushort(port))
}

func P2PClose() {
	C.p2p_close()
}

func P2PConnect(id uint64, ip string, port uint16) {
	C.p2p_connect(C.uint64_t(id), C.CString(ip), C.ushort(port))
}

func P2PShutdown(session uint32) {
	C.p2p_shutdown(C.uint(session))
}

func P2PSessionRtt(session uint32) uint32 {
	r := C.p2p_kcp_rxrtt(C.uint32_t(session))
	return uint32(r)
}

func P2PSessionSendBufferCount(session uint32) uint32 {
	r := C.p2p_kcp_nsndbuf(C.uint(session))
	return uint32(r)
}

func P2PCacheSize() uint64 {
	r := C.p2p_cache_size()
	return uint64(r)
}

func P2PSend(session uint32, data []byte) {

	pendingSendBuffer := P2PSessionSendBufferCount(session)
	const  maxSendBuffer = 10240

	if pendingSendBuffer > maxSendBuffer {
		Logger.Debugf("session kcp send queue over 10240 drop this message,session id:%v pendingSendBuffer:%v ", session, pendingSendBuffer)
		return
	}

	const maxSize = 64 * 1024
	totalLen := len(data)

	curPos := 0
	for curPos < totalLen {
		sendSize := totalLen - curPos
		if sendSize > maxSize {
			sendSize = maxSize
		}
		C.p2p_send(C.uint(session), unsafe.Pointer(&data[curPos]), C.uint(sendSize))
		curPos += sendSize
	}

}
