package definition

import (
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/db"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/abi"
	"github.com/zenon-network/go-zenon/vm/constants"
)

const (
	jsonHtlc = `
	[
		{"type":"function","name":"CreateHtlc", "inputs":[
			{"name":"hashLocked","type":"address"},
			{"name":"expirationTime","type":"int64"},
			{"name":"hashType","type":"uint8"},
			{"name":"keyMaxSize","type":"uint8"},
			{"name":"hashLock","type":"bytes"}
		]},
		{"type":"function","name":"ReclaimHtlc","inputs":[
			{"name":"id","type":"hash"}
		]},
		{"type":"function","name":"UnlockHtlc","inputs":[
			{"name":"id","type":"hash"},
			{"name":"preimage","type":"bytes"}
		]},

		{"type":"variable","name":"htlcInfo","inputs":[
			{"name":"timeLocked","type":"address"},
			{"name":"hashLocked","type":"address"},
			{"name":"tokenStandard","type":"tokenStandard"},
			{"name":"amount","type":"uint256"},
			{"name":"expirationTime", "type":"int64"},
			{"name":"hashType","type":"uint8"},
			{"name":"keyMaxSize","type":"uint8"},
			{"name":"hashLock","type":"bytes"}
		]}
	]`

	CreateHtlcMethodName  = "CreateHtlc"
	ReclaimHtlcMethodName = "ReclaimHtlc"
	UnlockHtlcMethodName  = "UnlockHtlc"

	// re: reclaim vs revoke
	// some other embedded contracts have "revoke" methods
	// indicating an action which invalidates an entry and returns funds
	// for htlcs, we invalidate unlocking via preimage as soon as soon as the expiration time arrives
	// however the funds still sit in the contract and exist as an entry, so we use "reclaim"

	variableNameHtlcInfo = "htlcInfo"
)

const (
	HashTypeSHA3 uint8 = iota
	HashTypeSHA256
)

var HashTypeDigestSizes = map[uint8]uint8{
	HashTypeSHA3:   32,
	HashTypeSHA256: 32,
}

const (
	LockTypeTime uint8 = iota + 2
	LockTypeHash
)

var (
	ABIHtlc = abi.JSONToABIContract(strings.NewReader(jsonHtlc))

	htlcInfoKeyPrefix = []byte{1}
	timeLockKeyPrefix = []byte{LockTypeTime}
	hashLockKeyPrefix = []byte{LockTypeHash}
)

type CreateHtlcParam struct {
	HashLocked     types.Address `json:"hashLocked"`
	ExpirationTime int64         `json:"expirationTime"`
	HashType       uint8         `json:"hashType"`
	KeyMaxSize     uint8         `json:"keyMaxSize"`
	HashLock       []byte        `json:"hashLock"`
}

type HtlcInfo struct {
	Id             types.Hash               `json:"id"`
	TimeLocked     types.Address            `json:"timeLocked"`
	HashLocked     types.Address            `json:"hashLocked"`
	TokenStandard  types.ZenonTokenStandard `json:"tokenStandard"`
	Amount         *big.Int                 `json:"amount"`
	ExpirationTime int64                    `json:"expirationTime"`
	HashType       uint8                    `json:"hashType"`
	KeyMaxSize     uint8                    `json:"keyMaxSize"`
	HashLock       []byte                   `json:"hashLock"`
}

type UnlockHtlcParam struct {
	Id       types.Hash
	Preimage []byte
}

func (entry *HtlcInfo) Save(context db.DB) error {
	data, err := ABIHtlc.PackVariable(
		variableNameHtlcInfo,
		entry.TimeLocked,
		entry.HashLocked,
		entry.TokenStandard,
		entry.Amount,
		entry.ExpirationTime,
		entry.HashType,
		entry.KeyMaxSize,
		entry.HashLock,
	)
	if err != nil {
		return err
	}
	return context.Put(getHtlcInfoKey(entry.Id), data)
}
func (entry *HtlcInfo) Delete(context db.DB) error {
	return context.Delete(getHtlcInfoKey(entry.Id))
}

func getHtlcInfoKey(hash types.Hash) []byte {
	return common.JoinBytes(htlcInfoKeyPrefix, hash.Bytes())
}
func isHtlcInfoKey(key []byte) bool {
	return key[0] == htlcInfoKeyPrefix[0]
}

func unmarshalHtlcInfoKey(key []byte) (*types.Hash, error) {
	if !isHtlcInfoKey(key) {
		return nil, errors.Errorf("invalid key! Not htcl info key")
	}
	h := new(types.Hash)
	err := h.SetBytes(key[1:])
	if err != nil {
		return nil, err
	}

	return h, nil
}

func parseHtlcInfo(key, data []byte) (*HtlcInfo, error) {
	if len(data) > 0 {
		info := new(HtlcInfo)
		if err := ABIHtlc.UnpackVariable(info, variableNameHtlcInfo, data); err != nil {
			return nil, err
		}
		id, err := unmarshalHtlcInfoKey(key)
		if err != nil {
			return nil, err
		}
		info.Id = *id
		return info, nil
	} else {
		return nil, constants.ErrDataNonExistent
	}
}
func GetHtlcInfo(context db.DB, id types.Hash) (*HtlcInfo, error) {
	key := getHtlcInfoKey(id)
	if data, err := context.Get(key); err != nil {
		return nil, err
	} else {
		return parseHtlcInfo(key, data)
	}
}

type HtlcRef struct {
	LockType uint8
	Unlocker types.Address
	Id       types.Hash
}

func (entry *HtlcRef) Save(context db.DB) error {
	// All of the information for a ref is stored in its key
	// We store a nonempty marker []byte{1} for its value
	// DB.Delete() is just a Put with []byte{}, not a true delete
	// So parseHtlcInfo has a len(data) > 0 check

	return context.Put(getHtlcRefKey([]byte{entry.LockType}, entry.Unlocker, entry.Id), []byte{1})
}

func (entry *HtlcRef) Delete(context db.DB) error {
	return context.Delete(getHtlcRefKey([]byte{entry.LockType}, entry.Unlocker, entry.Id))
}

func getHtlcRefKey(lockTypePrefix []byte, unlocker types.Address, id types.Hash) []byte {
	return common.JoinBytes(lockTypePrefix, unlocker.Bytes(), id.Bytes())
}

func isHtlcRefKey(key []byte) bool {
	return key[0] == timeLockKeyPrefix[0] || key[0] == hashLockKeyPrefix[0]
}

func unmarshalHtlcRefKey(key []byte) (uint8, *types.Hash, *types.Address, error) {
	if !isHtlcRefKey(key) {
		return 0, nil, nil, errors.Errorf("invalid key! Not htlc ref key")
	}
	h := new(types.Hash)
	err := h.SetBytes(key[1+types.AddressSize:])
	if err != nil {
		return 0, nil, nil, err
	}

	addr := new(types.Address)
	err = addr.SetBytes(key[1 : 1+types.AddressSize])
	if err != nil {
		return 0, nil, nil, err
	}

	return key[0], h, addr, nil
}

func parseHtlcRef(key []byte, data []byte) (*HtlcRef, error) {
	if len(data) > 0 {
		ref := new(HtlcRef)
		locktype, id, unlocker, err := unmarshalHtlcRefKey(key)
		if err != nil {
			return nil, err
		}
		ref.LockType = locktype
		ref.Unlocker = *unlocker
		ref.Id = *id
		return ref, nil
	} else {
		return nil, constants.ErrDataNonExistent
	}
}

func GetHtlcRefList(context db.DB, locktype uint8, unlocker types.Address) ([]*HtlcRef, error) {
	iterator := context.NewIterator(common.JoinBytes([]byte{locktype}, unlocker.Bytes()))
	defer iterator.Release()
	list := make([]*HtlcRef, 0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}

		// probably should refactor this and parseHtlcRef
		data, err := context.Get(iterator.Key())
		if err != nil {
			return nil, err
		}

		if ref, err := parseHtlcRef(iterator.Key(), data); err == nil {
			list = append(list, ref)
		} else if err == constants.ErrDataNonExistent {
			continue
		} else {
			return nil, err
		}
	}
	return list, nil
}
