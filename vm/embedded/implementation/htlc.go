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

func checkHtlc(param definition.CreateHtlcParam) error {

	if param.HashType != definition.HashTypeSHA3 && param.HashType != definition.HashTypeSHA256 {
		return constants.ErrInvalidHashType
	}

	if len(param.HashLock) != int(definition.HashTypeDigestSizes[param.HashType]) {
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

	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeTime,
		Unlocker: sendBlock.Address,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeHash,
		Unlocker: param.HashLocked,
	}

	common.DealWithErr(htlcInfo.Save(context.Storage()))
	common.DealWithErr(timelock.Save(context.Storage()))
	common.DealWithErr(hashlock.Save(context.Storage()))

	htlcLog.Debug("created new entry", "htlcInfo", htlcInfo)
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
		return nil, constants.ReclaimNotDue
	}

	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeTime,
		Unlocker: htlcInfo.TimeLocked,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeHash,
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

	var hashedPreimage []byte
	if htlcInfo.HashType == definition.HashTypeSHA3 {
		hashedPreimage = crypto.Hash(param.Preimage)
	} else if htlcInfo.HashType == definition.HashTypeSHA256 {
		hashedPreimage = crypto.HashSHA256(param.Preimage)
	} else {
		// shouldn't get here
	}

	if !bytes.Equal(hashedPreimage, htlcInfo.HashLock) {
		return nil, constants.ErrInvalidPreimage
	}

	timelock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeTime,
		Unlocker: htlcInfo.TimeLocked,
	}
	hashlock := definition.HtlcRef{
		Id:       htlcInfo.Id,
		LockType: definition.LockTypeHash,
		Unlocker: htlcInfo.HashLocked,
	}

	common.DealWithErr(htlcInfo.Delete(context.Storage()))
	common.DealWithErr(timelock.Delete(context.Storage()))
	common.DealWithErr(hashlock.Delete(context.Storage()))

	// TODO include preimage?
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
