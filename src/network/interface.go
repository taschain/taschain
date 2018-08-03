package network

type Message struct {
	Code uint32

	Body []byte
}

type Conn struct {
	Id   string
	Ip   string
	Port string
}

type MsgDigest []byte

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}

type Network interface {
	//Send message to the node which id represents.If self doesn't connect to the node,
	// resolve the kad net to find the node and then send the message
	Send(id string, msg Message) error

	//Send message to the node which id represents. If self doesn't connect to the ode,
	// send message to the guys which belongs to the same group with the node and they will rely the message to the node
	SendWithGroupRely(id string, groupId string,msg Message)error

	//Broadcast the message among the group
	Multicast(groupId string, msg Message) error

	//Send message to the neighborhood nodes
	TransmitToNeighbor(msg Message) error

	//Broadcast the message to all nodes
	Broadcast(msg Message,msgDigest MsgDigest) error

	//Return all connections self has
	ConnInfo() []Conn

	BuildGroupNet(groupId string, members []string)

	DissolveGroupNet(groupId string)

}
