package cli

// Wallet 账户struct
type wallet struct {
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
}
