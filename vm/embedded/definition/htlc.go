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

var (
	ABIHtlc = abi.JSONToABIContract(strings.NewReader(jsonHtlc))

	htlcInfoKeyPrefix = []byte{1}
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
