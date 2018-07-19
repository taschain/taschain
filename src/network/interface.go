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

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}

type Server interface {

	Send(targetId string, msg Message) error

	Multicast(groupId string, msg Message) error

	Broadcast(msg Message) error

	ConnInfo() []Conn
}
