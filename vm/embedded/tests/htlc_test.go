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
// TODO test create htlc token amount must be positive, how?? it is in the ValidateSendBlock

func TestHtlc_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()

	//
	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
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

	// TODO verify hashlock is correct

	expectedId := types.HexToHashPanic("2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489",
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
[
	{
		"id": "2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489",
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
`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
[
	{
		"id": "2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489",
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

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
}

func TestHtlc_reclaim(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
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

	expectedId := types.HexToHashPanic("5c967ef4862a0fd08e76c03c477f5b70ac79efbddcec0b5d273daa244e296f9e")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "5c967ef4862a0fd08e76c03c477f5b70ac79efbddcec0b5d273daa244e296f9e",
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
[
	{
		"id": "5c967ef4862a0fd08e76c03c477f5b70ac79efbddcec0b5d273daa244e296f9e",
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
`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `
[
	{
		"id": "5c967ef4862a0fd08e76c03c477f5b70ac79efbddcec0b5d273daa244e296f9e",
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

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)

	common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
	common.Json(htlcApi.GetHtlcInfosByHashLockedAddress(g.User2.Address, 0, 10)).Equals(t, `[]`)
}

func TestHtlc_unlock_access(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	expectedId := types.HexToHashPanic("2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489")

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

func TestHtlc_reclaim_access(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
		),
		TokenStandard: types.QsrTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertMomentumsTo(40)

	expectedId := types.HexToHashPanic("5c967ef4862a0fd08e76c03c477f5b70ac79efbddcec0b5d273daa244e296f9e")

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

	preimage := []byte("pump it zennie chan")
	nonexistentId := types.HexToHashPanic("2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489")

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

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	htlcId := types.HexToHashPanic("2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489")

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

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user 1 creates an htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertMomentumsTo(40)

	htlcId := types.HexToHashPanic("2ac372d2d9d1dc8679519225d107bff319a72231b1189be2840b5381d0834489")

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

	preimage := []byte("pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user tries to create expired htlc
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp-300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
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

	preimage := []byte("pump it zennie chan pump it zennie chan pump it zennie chan pump it zennie chan")
	lock := crypto.Hash(preimage)

	// user1 creates htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,             // hashlocked
			int64(genesisTimestamp+300), // expiration time
			uint8(0),                    // hash type
			uint8(32),                   // max preimage size
			lock,                        // hashlock
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	//htlcApi := embedded.NewHtlcApi(z)
	//common.Json(htlcApi.GetHtlcInfosByTimeLockedAddress(g.User1.Address, 0, 10)).Equals(t, `[]`)
	//TODO get expectedId dynamically everywhere

	expectedId := types.HexToHashPanic("eb5c933403d44e79ad803d8f4a49505f8225a832717687b640e07157b42a6036")

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