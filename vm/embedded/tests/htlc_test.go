package tests

import (
	"math/big"
	"testing"

	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/rpc/api/embedded"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"

	g "github.com/zenon-network/go-zenon/chain/genesis/mock"
	"github.com/zenon-network/go-zenon/chain/nom"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/crypto"
	"github.com/zenon-network/go-zenon/zenon/mock"
)

const (

	// why not just make this pubic in mock genesis?
	genesisTimestamp = 1000000000
)

// TODO test logs
// TODO don't test against strings, construct expected objects, marshal and compare
// TODO hide hashes
// TODO test fixtures, and helper methods

// TODO refactor tests
// most tedious testing would be ensuring invalid attempts don't alter validity
// many different permutations...
// would be overkill; I am confident without it
// can we check that DB state is unchanged?

// TODO generic activate spork helper function
func activateHtlc(z mock.MockZenon) {
	sporkAPI := embedded.NewSporkApi(z)
	z.InsertSendBlock(&nom.AccountBlock{
		Address:   g.Spork.Address,
		ToAddress: types.SporkContract,
		Data: definition.ABISpork.PackMethodPanic(definition.SporkCreateMethodName,
			"spork-htlc",              // name
			"activate spork for htlc", // description
		),
	}, nil, mock.SkipVmChanges)
	z.InsertNewMomentum()

	sporkList, _ := sporkAPI.GetAll(0, 10)
	id := sporkList.List[0].Id

	z.InsertSendBlock(&nom.AccountBlock{
		Address:   g.Spork.Address,
		ToAddress: types.SporkContract,
		Data: definition.ABISpork.PackMethodPanic(definition.SporkActivateMethodName,
			id, // id
		),
	}, nil, mock.SkipVmChanges)
	z.InsertNewMomentum()
	types.HtlcSpork.SporkId = id
	types.ImplementedSporksMap[id] = true
	z.InsertMomentumsTo(20)
}

// TODO test create htlc token amount must be positive, how?? it is in the ValidateSendBlock
// have tested it and gotten the correct error but can't do it through the expector
// move check from ValidateSendBlock to ReceiveBlock?
//func TestHtlc_zero(t *testing.T) {
//	z := mock.NewMockZenon(t)
//	defer z.StopPanic()
// 	activateHtlc(z)
//
//	preimage := []byte("pump it zennie chan")
//	lock := crypto.HashSHA256(preimage)
//
//	// user 1 creates an htlc for user 2
//	defer z.CallContract(&nom.AccountBlock{
//		Address:   g.User1.Address,
//		ToAddress: types.HtlcContract,
//		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
//			g.User2.Address,                // hashlocked
//			int64(genesisTimestamp+300),    // expiration time
//			uint8(definition.HashTypeSHA3), // hash type
//			uint8(32),                      // max preimage size
//			lock,                           // hashlock
//		),
//		TokenStandard: types.ZnnTokenStandard,
//		Amount:        big.NewInt(0),
//	}).Error(t, constants.ErrInvalidTokenOrAmount)
//	z.InsertNewMomentum()
//	z.InsertNewMomentum()
//}

func TestHtlc_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	// TODO make this a helper function
	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 11990*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 10*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

	expectedId := types.HexToHashPanic("582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
}
`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 1,
	"list": [
		{
			"id": "582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 0,
			"keyMaxSize": 32,
			"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
		}
	]
}
`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 1,
	"list": [
		{
			"id": "582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 0,
			"keyMaxSize": 32,
			"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
		}
	]
}
`)

	// user 1 tries to reclaim unexpired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			expectedId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ReclaimNotDue)
	z.InsertNewMomentum()

	// user 2 tries to unlock with wrong preimage
	wrong_preimage := []byte("pump it quasar chan")
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId,     // entry id
			wrong_preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

	// user2 unlocks with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	autoreceive(t, z, g.User2.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 11990*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8010*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
}

func TestHtlc_reclaim(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.QsrTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertMomentumsTo(40)

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 12000*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 119990*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 10*g.Zexp)

	// TODO verify hashlock is correct

	expectedId := types.HexToHashPanic("bfe77dadfce1d2a456a7338c30f6890bb22d4f6f217448459312ef0b23f45554")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "bfe77dadfce1d2a456a7338c30f6890bb22d4f6f217448459312ef0b23f45554",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1qsrxxxxxxxxxxxxxmrhjll",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
}
`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 1,
	"list": [
		{
			"id": "bfe77dadfce1d2a456a7338c30f6890bb22d4f6f217448459312ef0b23f45554",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1qsrxxxxxxxxxxxxxmrhjll",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 0,
			"keyMaxSize": 32,
			"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
		}
	]
}
`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 1,
	"list": [
		{
			"id": "bfe77dadfce1d2a456a7338c30f6890bb22d4f6f217448459312ef0b23f45554",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1qsrxxxxxxxxxxxxxmrhjll",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 0,
			"keyMaxSize": 32,
			"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
		}
	]
}
`)

	// user2 tries to unlock expired with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrExpired)
	z.InsertNewMomentum()

	// user 1 reclaims expired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			expectedId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	autoreceive(t, z, g.User1.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 12000*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
}

func TestHtlc_unlock_access(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	expectedId := types.HexToHashPanic("582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc")

	// user 1 tries to unlock with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrPermissionDenied)
	z.InsertNewMomentum()

	// user 3 tries to unlock with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrPermissionDenied)
	z.InsertNewMomentum()

	// user 2 unlocks with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

}

// TODO everyone tries to claim unexpired ?
func TestHtlc_reclaim_access(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.QsrTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertMomentumsTo(40)

	expectedId := types.HexToHashPanic("bfe77dadfce1d2a456a7338c30f6890bb22d4f6f217448459312ef0b23f45554")

	// user 2 tries to reclaim expired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			expectedId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrPermissionDenied)
	z.InsertNewMomentum()

	// user 3 tries to reclaim expired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			expectedId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrPermissionDenied)
	z.InsertNewMomentum()

	// user 1 reclaims expired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			expectedId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

}

func TestHtlc_nonexistent(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	nonexistentId := types.HexToHashPanic("582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc")

	// get htlcinfo rpc nonexistent
	common.Json(htlcApi.GetHtlcInfoById(nonexistentId)).Error(t, constants.ErrDataNonExistent)

	// unlock nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			nonexistentId, // entry id
			preimage,      // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()

	// reclaim nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			nonexistentId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()
}

func TestHtlc_nonexistent_after_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	htlcId := types.HexToHashPanic("582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc")

	// user2 unlocks with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			htlcId,   // entry id
			preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	// get htlcinfo rpc nonexistent
	common.Json(htlcApi.GetHtlcInfoById(htlcId)).Error(t, constants.ErrDataNonExistent)

	// unlock nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			htlcId,   // entry id
			preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()

	// reclaim nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			htlcId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()
}

func TestHtlc_nonexistent_after_reclaim(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertMomentumsTo(40)

	htlcId := types.HexToHashPanic("582052b563f7ed9a01c56248e58c938ac8826d935d88b170fef83b946f6a5bcc")

	// user1 reclaims expired
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			htlcId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	// get htlcinfo rpc nonexistent
	common.Json(htlcApi.GetHtlcInfoById(htlcId)).Error(t, constants.ErrDataNonExistent)

	// unlock nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			htlcId,   // entry id
			preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()

	// reclaim nonexistent
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.ReclaimHtlcMethodName,
			htlcId, // entry id
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrDataNonExistent)
	z.InsertNewMomentum()
}

func TestHtlc_create_expired(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user tries to create expired htlc
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp-300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, constants.ErrInvalidExpirationTime)
	z.InsertNewMomentum()

}

// test unlock htlc with ErrInvalidPreimage
func TestHtlc_unlock_long_preimage(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan pump it zennie chan pump it zennie chan pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user1 creates htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lock,                           // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	//htlcApi := embedded.NewHtlcApi(z)
	//common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)
	//TODO get expectedId dynamically everywhere

	expectedId := types.HexToHashPanic("00f92900a50db2b9c120b7d5934ad2fa9e30417fb120000970916a5b8c7dcbdb")

	// user2 tries to unlock with oversized preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

}

// SHA256 testing
// blackbox testing principles would dictate that I run the same set of tests
// as above with each different hash type
// But until we have parameterized tests I will just add streamlined higher value tests
// happy path and making sure sha3 preimage can't unlock sha256 hashlock
// and vice versa

func TestHtlc_unlockSHA256(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	lock := crypto.HashSHA256(preimage)

	// user 1 creates an htlc for user 2 using sha256 locktype
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                  // hashlocked
			int64(genesisTimestamp+300),      // expiration time
			uint8(definition.HashTypeSHA256), // hash type
			uint8(32),                        // max preimage size
			lock,                             // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 11990*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 10*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

	//common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	expectedId := types.HexToHashPanic("111bc3d0ecf5b1e00f0f464a990969c7ea99cf019cb99b02552472310e3a1884")

	//common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "111bc3d0ecf5b1e00f0f464a990969c7ea99cf019cb99b02552472310e3a1884",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 1,
	"keyMaxSize": 32,
	"hashLock": "0M84vVCA5ZZORK3DgSj2Q7kV9oMGqKDsPDgELO5il1o="
}
`)

	// user 2 tries to unlock with wrong preimage
	wrong_preimage := []byte("pump it quasar chan")
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId,     // entry id
			wrong_preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

	// user2 unlocks with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	autoreceive(t, z, g.User2.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 11990*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8010*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 0,
	"list": []
}`)
}

func TestHtlc_hashType(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	activateHtlc(z)

	preimage := []byte("pump it zennie chan")
	sha3lock := crypto.Hash(preimage)
	sha256lock := crypto.HashSHA256(preimage)

	// user 1 creates an htlc for user 2 using sha3 locktype with sha256 hashlock
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			sha256lock,                     // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	// user 1 creates an htlc for user 2 using sha256
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                  // hashlocked
			int64(genesisTimestamp+300),      // expiration time
			uint8(definition.HashTypeSHA256), // hash type
			uint8(32),                        // max preimage size
			sha3lock,                         // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
{
	"count": 2,
	"list": [
		{
			"id": "01f63fdcf903372256329648e2d722976f2f87f2e3ab16bb9a443f3eeb82e226",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 1,
			"keyMaxSize": 32,
			"hashLock": "t4Ra3NQe7E5Pocx1qGgBSBG1dZQsbkpyVRvAH2NwVjQ="
		},
		{
			"id": "e57011f283da5485d56287e520f42d4b67da5fa4276cbdda7c7139119419beeb",
			"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
			"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
			"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
			"amount": 1000000000,
			"expirationTime": 1000000300,
			"hashType": 0,
			"keyMaxSize": 32,
			"hashLock": "0M84vVCA5ZZORK3DgSj2Q7kV9oMGqKDsPDgELO5il1o="
		}
	]
}
`)

	expectedId1 := types.HexToHashPanic("01f63fdcf903372256329648e2d722976f2f87f2e3ab16bb9a443f3eeb82e226")
	expectedId2 := types.HexToHashPanic("e57011f283da5485d56287e520f42d4b67da5fa4276cbdda7c7139119419beeb")

	// user 2 cannot unlock sha3 lock
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId1, // entry id
			preimage,    // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

	// user 2 cannot unlock sha256 lock
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId2, // entry id
			preimage,    // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

}
