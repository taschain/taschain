package network
import "C"
import "fmt"
import "time"
//export OnP2PRecved
func OnP2PRecved(id uint64, session uint32, data []byte) {
	//fmt.Printf("%v %v %v %v\n", "OnP2PRecved", id, session, len(data))
	start := time.Now()

	Network.netCore.OnRecved(id, session, data)

	diff := time.Now().Sub(start)
	if diff > 500 *time.Millisecond {
		fmt.Printf("OnP2PRecved timeout:%v\n", diff)
	}
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
	Network.netCore.OnAccepted(id, session, p2p_type)
}

//export OnP2PConnected
func OnP2PConnected(id uint64, session uint32, p2p_type uint32) {
	start := time.Now()

	fmt.Printf("%v %v %v %v\n", "OnP2PConnected", id, session, p2p_type)
	Network.netCore.OnConnected(id, session, p2p_type)
	diff := time.Now().Sub(start)
	if diff > 500 *time.Millisecond {
		fmt.Printf("OnP2PConnected timeout:%v\n", diff)
	}

}

//export OnP2PDisconnected
func OnP2PDisconnected(id uint64, session uint32, p2p_code uint32) {
	start := time.Now()
	fmt.Printf("%v %v %v %v\n", "OnP2PDisconnected", id, session, p2p_code)
	Network.netCore.OnDisconnected(id, session, p2p_code)
	diff := time.Now().Sub(start)
	if diff > 500 *time.Millisecond {
		fmt.Printf("OnP2PDisconnected timeout:%v\n", diff)
	}
}
