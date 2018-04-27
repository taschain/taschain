package contract

import "common"

/*
**  Creator: pxf
**  Date: 2018/4/25 下午5:59
**  Description: 
*/

const (
	VOTE_ADDR_POOL_CODE = `6060604052341561000f57600080fd5b610b1a8061001e6000396000f30060606040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633ccc1f53146100725780636844dbea146100c35780639ab7d4e21461013a578063b5a8f068146101a4578063e6db7e281461020e575b600080fd5b341561007d57600080fd5b6100a9600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610292565b604051808215151515815260200191505060405180910390f35b34156100ce57600080fd5b6100fa600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506103e5565b604051808467ffffffffffffffff1667ffffffffffffffff1681526020018381526020018260001916600019168152602001935050505060405180910390f35b341561014557600080fd5b61014d6104ef565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b83811015610190578082015181840152602081019050610175565b505050509050019250505060405180910390f35b34156101af57600080fd5b6101b76106ce565b6040518080602001828103825283818151815260200191508051906020019060200280838360005b838110156101fa5780820151818401526020810190506101df565b505050509050019250505060405180910390f35b341561021957600080fd5b610278600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803567ffffffffffffffff1690602001909190803567ffffffffffffffff16906020019091908035600019169060200190919050506108ad565b604051808215151515815260200191505060405180910390f35b6000806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600080820160006101000a81549067ffffffffffffffff02191690556000820160086101000a81549067ffffffffffffffff021916905560018201600090555050600090505b6001805490508110156103df578273ffffffffffffffffffffffffffffffffffffffff1660018281548110151561034c57fe5b906000526020600020900160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156103d2576001818154811015156103a357fe5b906000526020600020900160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690555b8080600101915050610319565b50919050565b60008060008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160009054906101000a900467ffffffffffffffff1692506000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160089054906101000a900467ffffffffffffffff1667ffffffffffffffff1691506000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206001015490509193909250565b6104f7610a38565b600080600082816105089190610a4c565b50600090505b600180549050811015610673574360008060018481548110151561052e57fe5b906000526020600020900160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160089054906101000a900467ffffffffffffffff1667ffffffffffffffff161415610666578180548060010182816105cf9190610a4c565b916000526020600020900160008060006001868154811015156105ee57fe5b906000526020600020900160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010154909190915090600019169055505b808060010191505061050e565b818054806020026020016040519081016040528092919081815260200182805480156106c257602002820191906000526020600020905b815460001916815260200190600101908083116106aa575b50505050509250505090565b6106d6610a38565b600080600082816106e79190610a4c565b50600090505b600180549050811015610852574360008060018481548110151561070d57fe5b906000526020600020900160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160009054906101000a900467ffffffffffffffff1667ffffffffffffffff161415610845578180548060010182816107ae9190610a4c565b916000526020600020900160008060006001868154811015156107cd57fe5b906000526020600020900160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060010154909190915090600019169055505b80806001019150506106ed565b818054806020026020016040519081016040528092919081815260200182805480156108a157602002820191906000526020600020905b81546000191681526020019060010190808311610889575b50505050509250505090565b6000836000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160006101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550826000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160086101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550816000808773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206001018160001916905550600180548060010182816109dd9190610a78565b9160005260206000209001600087909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505060019050949350505050565b602060405190810160405280600081525090565b815481835581811511610a7357818360005260206000209182019101610a729190610aa4565b5b505050565b815481835581811511610a9f57818360005260206000209182019101610a9e9190610ac9565b5b505050565b610ac691905b80821115610ac2576000816000905550600101610aaa565b5090565b90565b610aeb91905b80821115610ae7576000816000905550600101610acf565b5090565b905600a165627a7a72305820fad53be8f4c57d7b6377efccef96fe471baa7e950a0f221c0c818a966eb8918f0029`
	VOTE_ADDR_POOL_ABI = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVote","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"addr","type":"address"}],"name":"getVoteAddr","outputs":[{"name":"StatBlock","type":"uint64"},{"name":"EffectBlock","type":"uint64"},{"name":"TxHash","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getCurrentEffectVoteHashs","outputs":[{"name":"","type":"bytes32[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getCurrentStatVoteHashs","outputs":[{"name":"","type":"bytes32[]"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"},{"name":"sb","type":"uint64"},{"name":"eb","type":"uint64"},{"name":"h","type":"bytes32"}],"name":"addVote","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
)

type VoteAddrPool struct {
	BoundContract
	ctx *CallContext
}

type VoteAddr struct {
	StatBlock uint64
	EffectBlock uint64
	TxHash common.Hash
	Addr common.Address
}

func NewVoteAddrPool(ctx *CallContext, bc *BoundContract) *VoteAddrPool {
	return &VoteAddrPool{
		BoundContract: *bc,
		ctx: ctx,
	}
}

func (vp *VoteAddrPool) AddVote(v *VoteAddr) (bool, error) {
	if ret, err := vp.ResultCall(vp.ctx, func() interface{} {
		return new(bool)
	}, NewCallOpt(nil, "addVote", v.Addr, v.StatBlock, v.EffectBlock, v.TxHash)); err != nil {
		return false, err
	} else {
		return *(ret.(*bool)), nil
	}
}

func (vp *VoteAddrPool) RemoveVote(address common.Address) (bool, error) {
	if ret, err := vp.ResultCall(vp.ctx, func() interface{} {
		return new(bool)
	}, NewCallOpt(nil, "removeVote", address)); err != nil {
		return false, err
	} else {
		return *(ret.(*bool)), nil
	}
}

func (vp *VoteAddrPool) GetCurrentStatVoteHashes() ([]common.Hash, error) {
	if ret, err := vp.ResultCall(vp.ctx, func() interface{} {
		return new([][32]byte)
	}, NewCallOpt(nil, "getCurrentStatVoteHashs")); err != nil {
		return nil, err
	} else {
		bytes := *ret.(*[][32]byte)
		hs := make([]common.Hash, 0)
		for _, b := range bytes {
			hs = append(hs, common.BytesToHash(b[:]))
		}
		return hs, nil
	}
}

func (vp *VoteAddrPool) GetCurrentEffectVoteHashes() ([]common.Hash, error) {
	if ret, err := vp.ResultCall(vp.ctx, func() interface{} {
		return new([][32]byte)
	}, NewCallOpt(nil, "getCurrentEffectVoteHashs")); err != nil {
		return nil, err
	} else {
		bytes := *ret.(*[][32]byte)
		hs := make([]common.Hash, 0)
		for _, b := range bytes {
			hs = append(hs, common.BytesToHash(b[:]))
		}
		return hs, nil
	}
}

func (vp *VoteAddrPool) GetVoteAddr(address common.Address) (*VoteAddr, error) {
	if ret, err := vp.ResultCall(vp.ctx, func() interface{} {
		return new(VoteAddr)
	}, NewCallOpt(nil, "getVoteAddr", address)); err != nil {
		return nil, err
	} else {
		return ret.(*VoteAddr), nil
	}
}