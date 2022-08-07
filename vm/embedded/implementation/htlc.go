package implementation

import (
	"bytes"

	"github.com/zenon-network/go-zenon/chain/nom"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/crypto"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
	"github.com/zenon-network/go-zenon/vm/vm_context"
)

var (
	htlcLog = common.EmbeddedLogger.New("contract", "htlc")
)

// TODO allow other hashtypes
// TODO: make sure hashlock is valid given hashtype
func checkHtlc(param definition.CreateHtlcParam) error {

	if param.HashType != 0 {
		return constants.ErrInvalidHashType
	}

	if len(param.HashLock) != 32 {
		return constants.ErrInvalidHashDigest
	}

	return nil
}

type CreateHtlcMethod struct {
	MethodName string
}

func (p *CreateHtlcMethod) GetPlasma(plasmaTable *constants.PlasmaTable) (uint64, error) {
	return plasmaTable.EmbeddedSimple, nil
}
func (p *CreateHtlcMethod) ValidateSendBlock(block *nom.AccountBlock) error {
	var err error

	param := new(definition.CreateHtlcParam)

	if err := definition.ABIHtlc.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return constants.ErrUnpackError
	}

	if err = checkHtlc(*param); err != nil {
		return err
	}

	// can't create empty htlcs
	if block.Amount.Sign() == 0 {
		return constants.ErrInvalidTokenOrAmount
	}

	block.Data, err = definition.ABIHtlc.PackMethod(p.MethodName,
		param.HashLocked,
		param.ExpirationTime,
		param.HashType,
		param.KeyMaxSize,
		param.HashLock,
	)
	return err
}
func (p *CreateHtlcMethod) ReceiveBlock(context vm_context.AccountVmContext, sendBlock *nom.AccountBlock) ([]*nom.AccountBlock, error) {
	if err := p.ValidateSendBlock(sendBlock); err != nil {
		return nil, err
	}

	param := new(definition.CreateHtlcParam)
	err := definition.ABIHtlc.UnpackMethod(param, p.MethodName, sendBlock.Data)
	common.DealWithErr(err)

	momentum, err := context.GetFrontierMomentum()
	common.DealWithErr(err)

	// can't create htlc that is already expired
	// what other constraints do we want to put on expiration time?
	// e.g minumum duration?, max?
	// went with raw time as opposed to blockheight to help coordinate across chains
	// and to be the most flexible
	if param.ExpirationTime <= momentum.Timestamp.Unix() {
		return nil, constants.ErrInvalidExpirationTime
	}

	htlcInfo := definition.HtlcInfo{
		Id:             sendBlock.Hash,
		TimeLocked:     sendBlock.Address,
		HashLocked:     param.HashLocked,
		TokenStandard:  sendBlock.TokenStandard,
		Amount:         sendBlock.Amount,
		ExpirationTime: param.ExpirationTime,
		HashType:       param.HashType,
		KeyMaxSize:     param.KeyMaxSize,
		HashLock:       param.HashLock,
	}

	// TODO get rid of magic number locktypes in this file

	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{2},
		Unlocker: sendBlock.Address,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{3},
		Unlocker: param.HashLocked,
	}

	common.DealWithErr(htlcInfo.Save(context.Storage()))
	common.DealWithErr(timelock.Save(context.Storage()))
	common.DealWithErr(hashlock.Save(context.Storage()))

	// TODO
	htlcLog.Debug("created new entry", "htlcInfo", htlcInfo, "locker", sendBlock.Address)
	return nil, nil
}

type ReclaimHtlcMethod struct {
	MethodName string
}

func (p *ReclaimHtlcMethod) GetPlasma(plasmaTable *constants.PlasmaTable) (uint64, error) {
	return plasmaTable.EmbeddedWWithdraw, nil
}
func (p *ReclaimHtlcMethod) ValidateSendBlock(block *nom.AccountBlock) error {
	var err error
	param := new(types.Hash)

	if err := definition.ABIHtlc.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return constants.ErrUnpackError
	}

	if block.Amount.Sign() > 0 {
		return constants.ErrInvalidTokenOrAmount
	}

	block.Data, err = definition.ABIHtlc.PackMethod(p.MethodName, param)
	return err
}
func (p *ReclaimHtlcMethod) ReceiveBlock(context vm_context.AccountVmContext, sendBlock *nom.AccountBlock) ([]*nom.AccountBlock, error) {
	if err := p.ValidateSendBlock(sendBlock); err != nil {
		return nil, err
	}

	id := new(types.Hash)
	err := definition.ABIHtlc.UnpackMethod(id, p.MethodName, sendBlock.Data)
	common.DealWithErr(err)

	momentum, err := context.GetFrontierMomentum()
	common.DealWithErr(err)

	htlcInfo, err := definition.GetHtlcInfo(context.Storage(), *id)
	if err == constants.ErrDataNonExistent {
		return nil, err
	}
	common.DealWithErr(err)

	// only timelocked can reclaim
	if htlcInfo.TimeLocked != sendBlock.Address {
		return nil, constants.ErrPermissionDenied
	}

	// can only reclaim after the entry is expired
	if htlcInfo.ExpirationTime > momentum.Timestamp.Unix() {
		// TODO new constant for reclaim not due?
		return nil, constants.ReclaimNotDue
	}

	// TODO better to construct or fetch this?
	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{2},
		Unlocker: htlcInfo.TimeLocked,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{3},
		Unlocker: htlcInfo.HashLocked,
	}

	common.DealWithErr(htlcInfo.Delete(context.Storage()))
	common.DealWithErr(timelock.Delete(context.Storage()))
	common.DealWithErr(hashlock.Delete(context.Storage()))

	htlcLog.Debug("reclaimed htlc entry", "htlcInfo", htlcInfo)

	return []*nom.AccountBlock{
		{
			Address:       types.HtlcContract,
			ToAddress:     sendBlock.Address,
			BlockType:     nom.BlockTypeContractSend,
			Amount:        htlcInfo.Amount,
			TokenStandard: htlcInfo.TokenStandard,
			Data:          []byte{},
		},
	}, nil
}

type UnlockHtlcMethod struct {
	MethodName string
}

func (p *UnlockHtlcMethod) GetPlasma(plasmaTable *constants.PlasmaTable) (uint64, error) {
	return plasmaTable.EmbeddedWWithdraw, nil
}
func (p *UnlockHtlcMethod) ValidateSendBlock(block *nom.AccountBlock) error {
	var err error
	param := new(definition.UnlockHtlcParam)

	if err := definition.ABIHtlc.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return constants.ErrUnpackError
	}

	if block.Amount.Sign() > 0 {
		return constants.ErrInvalidTokenOrAmount
	}

	block.Data, err = definition.ABIHtlc.PackMethod(p.MethodName, param.Id, param.Preimage)
	return err
}
func (p *UnlockHtlcMethod) ReceiveBlock(context vm_context.AccountVmContext, sendBlock *nom.AccountBlock) ([]*nom.AccountBlock, error) {
	if err := p.ValidateSendBlock(sendBlock); err != nil {
		return nil, err
	}

	param := new(definition.UnlockHtlcParam)
	err := definition.ABIHtlc.UnpackMethod(param, p.MethodName, sendBlock.Data)
	common.DealWithErr(err)

	momentum, err := context.GetFrontierMomentum()
	common.DealWithErr(err)

	htlcInfo, err := definition.GetHtlcInfo(context.Storage(), param.Id)
	if err == constants.ErrDataNonExistent {
		return nil, err
	}
	common.DealWithErr(err)

	// only hashlocked can unlock
	if sendBlock.Address != htlcInfo.HashLocked {
		return nil, constants.ErrPermissionDenied
	}

	// can only unlock before expiration time
	if momentum.Timestamp.Unix() > htlcInfo.ExpirationTime {
		return nil, constants.ErrExpired
	}

	if len(param.Preimage) > int(htlcInfo.KeyMaxSize) {
		return nil, constants.ErrInvalidPreimage
	}

	// TODO support other hash types
	if !bytes.Equal(crypto.Hash(param.Preimage), htlcInfo.HashLock) {
		return nil, constants.ErrInvalidPreimage
	}

	// TODO fetch this?
	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{2},
		Unlocker: htlcInfo.TimeLocked,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: []byte{3},
		Unlocker: htlcInfo.HashLocked,
	}

	common.DealWithErr(htlcInfo.Delete(context.Storage()))
	common.DealWithErr(timelock.Delete(context.Storage()))
	common.DealWithErr(hashlock.Delete(context.Storage()))

	htlcLog.Debug("unlocked htlc entry", "htlcInfo", htlcInfo)

	return []*nom.AccountBlock{
		{
			Address:       types.HtlcContract,
			ToAddress:     sendBlock.Address,
			BlockType:     nom.BlockTypeContractSend,
			Amount:        htlcInfo.Amount,
			TokenStandard: htlcInfo.TokenStandard,
			Data:          []byte{},
		},
	}, nil
}
