package p2p

type Handler interface {
	HandlerMessage(code uint32, body []byte, sourceId string) ([]byte,error)
}

var chainHandler Handler

var consensusHandler Handler

func SetChainHandler(h Handler) {
	if h != nil {
		chainHandler = h
	}
}

func SetConsensusHandler(h Handler) {
	if h != nil {
		consensusHandler = h
	}
}
