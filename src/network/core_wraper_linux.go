package network
/*
#cgo LDFLAGS:-ldl

#include <dlfcn.h>
#include <stdio.h>
void* p2p_api(const char* api)
{
    static void* p2p_core = 0;
    if (p2p_core == 0)
    {
        p2p_core = dlopen("./p2p_core.so", RTLD_LAZY);
        if (p2p_core == 0){
        	printf("p2p_core load lib failed !\n");
        }
    }
    return (void*)dlsym(p2p_core, api);
}

#include <stdint.h>
#include <string.h>

extern void OnP2PRecved(uint64_t id, uint32_t session, _GoBytes_ data);
extern void OnP2PChecked(uint32_t type, _GoString_ private_ip, _GoString_ public_ip);
extern void OnP2PListened(_GoString_ ip, uint16_t port, uint64_t latency);
extern void OnP2PAccepted(uint64_t id, uint32_t session, uint32_t type);
extern void OnP2PConnected(uint64_t id, uint32_t session, uint32_t type);
extern void OnP2PDisconnected(uint64_t id, uint32_t session, uint32_t code);

typedef void(*p2p_recved)(uint64_t id, uint32_t session, char* data, uint32_t size);
typedef void(*p2p_checked)(uint32_t type, const char* private_ip, const char* public_ip);
typedef void(*p2p_listened)(const char* ip, uint16_t port, uint64_t latency);
typedef void(*p2p_accepted)(uint64_t id, uint32_t session, uint32_t type);
typedef void(*p2p_connected)(uint64_t id, uint32_t session, uint32_t type);
typedef void(*p2p_disconnected)(uint64_t id, uint32_t session, uint32_t code);

struct p2p_callback
{
    p2p_recved       recved;
    p2p_checked      checked;
    p2p_listened     listened;
    p2p_accepted     accepted;
    p2p_connected    connected;
    p2p_disconnected disconnected;
};

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

void p2p_config(uint64_t id)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
        struct p2p_callback callback = { 0 };
   		callback.recved = on_p2p_recved;
        callback.checked = on_p2p_checked;
        callback.listened = on_p2p_listened;
        callback.accepted = on_p2p_accepted;
        callback.connected = on_p2p_connected;
        callback.disconnected = on_p2p_disconnected;
        ((void(*)(uint64_t id, struct p2p_callback callback))api)(id, callback);
    }
}

void p2p_proxy(const char* ip, uint16_t port)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)(const char* ip, uint16_t port))api)(ip, port);
    }
}

void p2p_listen(const char* ip, uint16_t port)
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)(const char* ip, uint16_t port))api)(ip, port);
    }
}

void p2p_close()
{
    void* api = p2p_api(__FUNCTION__);
    if (api)
    {
    	((void(*)())api)();
	}
}

void p2p_connect(uint64_t id, const char* ip, uint16_t port)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint64_t id, const char* ip, uint16_t port))api)(id, ip, port);
	}
}

void p2p_shutdown(uint32_t session)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint32_t session))api)(session);
	}
}

void p2p_send(uint32_t session, const void* data, uint32_t size)
{
	void* api = p2p_api(__FUNCTION__);
	if (api)
    {
    	((void(*)(uint32_t session, const void* data, uint32_t size))api)(session, data, size);
	}
}
*/
import "C"
import (
	"unsafe"
)

func P2PConfig(id uint64) {
	C.p2p_config(C.ulong(id))
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
	C.p2p_connect(C.ulong(id), C.CString(ip), C.ushort(port))
}

func P2PShutdown(session uint32) {
	C.p2p_shutdown(C.uint(session))
}

func P2PSend(session uint32, data []byte) {
	C.p2p_send(C.uint(session), unsafe.Pointer(&data[0]), C.uint(len(data)))
}


