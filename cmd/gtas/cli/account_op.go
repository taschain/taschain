package cli

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/storage/tasdb"
	"os"
	"sync"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午3:21
**  Description:
 */

const accountUnLockTime = time.Second * 120

var encryptPrivateKey *common.PrivateKey
var encryptPublicKey *common.PublicKey

func init() {
	encryptPrivateKey = common.HexToSecKey("0x04b851c3551779125a588b2274cfa6d71604fe6ae1f0df82175bcd6e6c2b23d92a69d507023628b59c15355f3cbc0d8f74633618facd28632a0fb3e9cc8851536c4b3f1ea7c7fd3666ce8334301236c2437d9bed14e5a0793b51a9a6e7a4c46e70")
	pk := encryptPrivateKey.GetPubKey()
	encryptPublicKey = &pk
}

const (
	statusLocked   int8 = 0
	statusUnLocked      = 1
)
const DefaultPassword = "123"

type AccountManager struct {
	store    *tasdb.LDBDatabase
	accounts sync.Map

	unlockAccount *AccountInfo
	mu            sync.Mutex
}

//var AccountOp accountOp

type AccountInfo struct {
	Account
	Status       int8
	UnLockExpire time.Time
}

func (ai *AccountInfo) unlocked() bool {
	return time.Now().Before(ai.UnLockExpire) && ai.Status == statusUnLocked
}

func (ai *AccountInfo) resetExpireTime() {
	ai.UnLockExpire = time.Now().Add(accountUnLockTime)
}

type Account struct {
	Address  string
	Pk       string
	Sk       string
	Password string
	Miner    *MinerRaw
}

type MinerRaw struct {
	BPk   string
	BSk   string
	VrfPk string
	VrfSk string
}

func dirExists(dir string) bool {
	f, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return f.IsDir()
}

func newAccountOp(ks string) (*AccountManager, error) {
	db, err := tasdb.NewLDBDatabase(ks, 128, 128)
	if err != nil {
		return nil, fmt.Errorf("new ldb fail:%v", err.Error())
	}
	return &AccountManager{
		store: db,
	}, nil
}

func InitAccountManager(keystore string, readyOnly bool) (accountOp, error) {
	//内部批量部署时，指定自动创建账号（只需创建一次）
	if readyOnly && !dirExists(keystore) {
		aop, err := newAccountOp(keystore)
		if err != nil {
			panic(err)
		}
		ret := aop.NewAccount(DefaultPassword, true)
		if !ret.IsSuccess() {
			fmt.Println(ret.Message)
			panic(ret.Message)
		}
		return aop, nil
	}

	//tmp := keystore
	//if readyOnly {
	//	if !dirExists(keystore) {
	//		os.Mkdir(keystore, os.ModePerm)
	//	}
	//	//要先将keystore目录拷贝一份，打开拷贝目录，否则gtas无法再打开该keystore
	//	tmp = fmt.Sprintf("tmp%c%v", os.PathSeparator, keystore)
	//	os.RemoveAll(tmp)
	//	if err := utility.Copy(keystore, tmp); err != nil {
	//		return nil, err
	//	}
	//}

	if aop, err := newAccountOp(keystore); err != nil {
		return nil, err
	} else {
		return aop, nil
	}

}

func (am *AccountManager) loadAccount(addr string) (*Account, error) {
	v, err := am.store.Get([]byte(addr))
	if err != nil {
		return nil, err
	}

	bs, err := encryptPrivateKey.Decrypt(rand.Reader, v)
	if err != nil {
		return nil, err
	}

	var acc = new(Account)
	err = json.Unmarshal(bs, acc)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func (am *AccountManager) storeAccount(account *Account) error {
	bs, err := json.Marshal(account)
	if err != nil {
		return err
	}

	ct, err := common.Encrypt(rand.Reader, encryptPublicKey, bs)
	if err != nil {
		return err
	}

	err = am.store.Put([]byte(account.Address), ct)
	return err
}

func (am *AccountManager) getFirstMinerAccount() *Account {
	iter := am.store.NewIterator()
	for iter.Next() {
		if ac, err := am.getAccountInfo(string(iter.Key())); err != nil {
			panic(fmt.Sprintf("getAccountInfo err,addr=%v,err=%v", string(iter.Key()), err.Error()))
		} else {
			if ac.Miner != nil {
				return &ac.Account
			}
		}
	}
	return nil
}

func (am *AccountManager) resetExpireTime(addr string) {
	acc, err := am.getAccountInfo(addr)
	if err != nil {
		return
	}
	acc.resetExpireTime()
}

func (am *AccountManager) getAccountInfo(addr string) (*AccountInfo, error) {
	var aci *AccountInfo
	if v, ok := am.accounts.Load(addr); ok {
		aci = v.(*AccountInfo)
	} else {
		acc, err := am.loadAccount(addr)
		if err != nil {
			return nil, err
		}
		aci = &AccountInfo{
			Account: *acc,
		}
		am.accounts.Store(addr, aci)
	}
	return aci, nil
}

func (am *AccountManager) currentUnLockedAddr() string {
	if am.unlockAccount != nil && am.unlockAccount.unlocked() {
		return am.unlockAccount.Address
	}
	return ""
}

func passwordSha(password string) string {
	return common.ToHex(common.Sha256([]byte(password)))
}

func (am *AccountManager) NewAccount(password string, miner bool) *Result {
	privateKey := common.GenerateKey("")
	pubkey := privateKey.GetPubKey()
	address := pubkey.GetAddress()

	account := &Account{
		Address:  address.Hex(),
		Pk:       pubkey.Hex(),
		Sk:       privateKey.Hex(),
		Password: passwordSha(password),
	}

	if miner {
		minerDO := model.NewSelfMinerDO(address)

		minerRaw := &MinerRaw{
			BPk:   minerDO.PK.GetHexString(),
			BSk:   minerDO.SK.GetHexString(),
			VrfPk: minerDO.VrfPK.GetHexString(),
			VrfSk: minerDO.VrfSK.GetHexString(),
		}
		account.Miner = minerRaw
	}
	if err := am.storeAccount(account); err != nil {
		return opError(err)
	}

	return opSuccess(address.Hex())
}

func (am *AccountManager) AccountList() *Result {
	iter := am.store.NewIterator()
	addrs := make([]string, 0)
	for iter.Next() {
		addrs = append(addrs, string(iter.Key()))
	}
	return opSuccess(addrs)
}

func (am *AccountManager) Lock(addr string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	aci.Status = statusLocked
	return opSuccess(nil)
}

func (am *AccountManager) UnLock(addr string, password string) *Result {
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if aci.Password != passwordSha(password) {
		return opError(ErrPassword)
	}
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.unlockAccount != nil && aci.Address != am.unlockAccount.Address {
		am.unlockAccount.Status = statusLocked
	}

	aci.Status = statusUnLocked
	aci.resetExpireTime()
	am.unlockAccount = aci

	return opSuccess(nil)
}

func (am *AccountManager) AccountInfo() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	aci.resetExpireTime()
	return opSuccess(&aci.Account)
}

func (am *AccountManager) DeleteAccount() *Result {
	addr := am.currentUnLockedAddr()
	if addr == "" {
		return opError(ErrUnlocked)
	}
	aci, err := am.getAccountInfo(addr)
	if err != nil {
		return opError(err)
	}
	if !aci.unlocked() {
		return opError(ErrUnlocked)
	}
	am.accounts.Delete(addr)
	am.store.Delete([]byte(addr))
	return opSuccess(nil)
}

func (am *AccountManager) Close() {
	am.store.Close()
}
