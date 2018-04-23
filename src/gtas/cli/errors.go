package cli

import "errors"

var (
	// ErrorBlockChainUninitialized 未初始化链
	ErrorBlockChainUninitialized = errors.New("should init blockchain module first")
	// ErrorP2PUninitialized 未初始化P2P模块。
	ErrorP2PUninitialized = errors.New("should init P2P module first")
	// ErrorGovUninitialized 未初始化共识模块。
	ErrorGovUninitialized = errors.New("should init Governance module first")
	// ErrorWalletsUninitialized
	ErrorWalletsUninitialized = errors.New("should load wallets from config")
)
