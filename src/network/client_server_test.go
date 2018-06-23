package network

import (
	"testing"
	"common"
	"network/p2p"
	"time"
	"math/rand"
)

func mockSeedServer() {
	common.InitConf("tas_test.ini")
	config := common.NewConfINIManager("tas_test.ini")
	common.GlobalConf.SetString("network", "private_key", "0x049b8eabeb44588b39b89a81f78a7c649e671d56a1d7c6ca35433a07f6be3aad7ebe765d1bb05a30d60ef3ec542495da77870ade6793388340428ea135dddf621008ea5707c18416934680568fe2ebb940358ba0c5281f55d710314b777f4f3ba1")
	config.SetString("network", "private_key", "0x049b8eabeb44588b39b89a81f78a7c649e671d56a1d7c6ca35433a07f6be3aad7ebe765d1bb05a30d60ef3ec542495da77870ade6793388340428ea135dddf621008ea5707c18416934680568fe2ebb940358ba0c5281f55d710314b777f4f3ba1")
	InitNetwork(&config,true)
}

func mockClientServer() {
	common.InitConf("tas_test.ini")
	config := common.NewConfINIManager("tas_test.ini")
	common.GlobalConf.SetString("network", "private_key", "0x04ecf26eff5b6bd5414068724e96907d582eab38787e00bfe3a08f44bcde2bf2db7180ede81b1ee58c5d9361178a649be2e6a6940cb23c686496be17310213632de3ca7043f0bfa159460507bc6ca46b85d62cdff41df36da53eeeb441a51b0d9e")
	config.SetString("network", "private_key", "0x04ecf26eff5b6bd5414068724e96907d582eab38787e00bfe3a08f44bcde2bf2db7180ede81b1ee58c5d9361178a649be2e6a6940cb23c686496be17310213632de3ca7043f0bfa159460507bc6ca46b85d62cdff41df36da53eeeb441a51b0d9e")
	InitNetwork(&config,false)
}

func TestServerNet(t *testing.T) {
	//seedId := "Qmdeh5r5kT2je77JNYKTsQi6ncckpLa9aFnr6xYQaGAxaw"
	clientId := "QmSDPsKnRfC4sbiQZLLNnGybt1gkG3GwgumnAHamu8zuwf"
	mockSeedServer()

	time.Sleep(15 * time.Second)
	for i := 0; i < 10; i++ {
		m := mockMessage()
		p2p.Server.SendMessage(m, clientId)
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(1 * time.Minute)
}

func TestClientNet(t *testing.T) {
	mockClientServer()
	time.Sleep(3 * time.Minute)
}

func mockMessage() p2p.Message {
	code := p2p.GROUP_INIT_MSG
	sign := []byte{1, 2, 3, 4, 5, 6, 7}

	r := rand.Intn(100000)
	body := make([]byte, r)
	for i := 0; i < r; i++ {
		body[i] = 8
	}
	m := p2p.Message{Code: code, Sign: sign, Body: body}
	return m
}
