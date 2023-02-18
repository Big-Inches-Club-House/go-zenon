package tests

import (
	"encoding/hex"
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
	genesisTimestamp = 1000000000
)

var (
	// iykyk
	preimageZ, _ = hex.DecodeString("b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634")
	preimageQ, _ = hex.DecodeString("d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee")

	preimageZlong = []byte("a718ee3fe739bd6435f0bd7bb4ee90e1deff2343372d92a04592c26a39b570a8")
)

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

func TestHtlc_zero(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg="invalid create - cannot create zero amount" module=embedded contract=htlc address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
`)
	activateHtlc(z)

	lock := crypto.HashSHA256(preimageZ)

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
		Amount:        big.NewInt(0),
	}).Error(t, constants.ErrInvalidTokenOrAmount)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, 12000*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)
}

func TestHtlc_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg="invalid reclaim - entry not expired" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz time=1000000220 expiration-time=1000000300
t=2001-09-09T01:50:30+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage=d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee
t=2001-09-09T01:50:40+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "Fd4QDoNykDbHp30bYIghQmK4OOcnATsGilLcpt7kV8s="
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
	wrong_preimage := preimageQ
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

}

func TestHtlc_reclaim(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:53:20+0000 lvl=dbug msg="invalid unlock - entry is expired" module=embedded contract=htlc id=dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx time=1000000400 expiration-time=1000000300
t=2001-09-09T01:53:30+0000 lvl=dbug msg=reclaimed module=embedded contract=htlc htlcInfo="&{Id:dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId := types.HexToHashPanic("dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1qsrxxxxxxxxxxxxxmrhjll",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "Fd4QDoNykDbHp30bYIghQmK4OOcnATsGilLcpt7kV8s="
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

}

func TestHtlc_proxy_unlock_default(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qrs2lpccnsneglhnnfwvlsj0qncnxjnwlfmjac preimage=d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee
t=2001-09-09T01:50:40+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
`)
	activateHtlc(z)

	preimage := preimageZ
	lock := crypto.Hash(preimage)

	// Check the default proxy unlock status
	common.Json(htlcApi.GetHtlcProxyUnlockStatus(g.User2.Address)).Equals(t, `true`)

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

	expectedId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

	// user 3 tries to unlock with wrong preimage
	wrong_preimage := preimageQ
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId,     // entry id
			wrong_preimage, // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	// user 3 unlocks with correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
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

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Error(t, constants.ErrDataNonExistent)

	autoreceive(t, z, g.User2.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-10)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, (8000+10)*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(g.User3.Address, types.ZnnTokenStandard, 1000*g.Zexp)
	z.ExpectBalance(g.User3.Address, types.QsrTokenStandard, 10000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)
}

func TestHtlc_deny_proxy_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg="deny proxy unlock" module=embedded contract=htlc address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx
t=2001-09-09T01:50:30+0000 lvl=dbug msg="invalid unlock - permission denied" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
t=2001-09-09T01:50:40+0000 lvl=dbug msg="invalid unlock - permission denied" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qrs2lpccnsneglhnnfwvlsj0qncnxjnwlfmjac
t=2001-09-09T01:50:50+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

	// user 2 disables proxy unlock
	defer z.CallContract(&nom.AccountBlock{
		Address:       g.User2.Address,
		ToAddress:     types.HtlcContract,
		Data:          definition.ABIHtlc.PackMethodPanic(definition.DenyHtlcProxyUnlockMethodName),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcProxyUnlockStatus(g.User2.Address)).Equals(t, `false`)

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
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Error(t, constants.ErrDataNonExistent)

	autoreceive(t, z, g.User2.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-10)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, (8000+10)*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)
}

func TestHtlc_allow_proxy_unlock(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:00f971921465de77c458075e3df560f7c52b2168d0ec3f253406dc8049e772ab TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[49 177 249 153 224 154 117 249 195 203 229 70 130 23 234 161 107 75 142 202 190 243 184 236 128 248 61 216 27 70 255 253]}"
t=2001-09-09T01:50:40+0000 lvl=dbug msg="deny proxy unlock" module=embedded contract=htlc address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx
t=2001-09-09T01:50:50+0000 lvl=dbug msg="allow proxy unlock" module=embedded contract=htlc address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx
t=2001-09-09T01:51:00+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qrs2lpccnsneglhnnfwvlsj0qncnxjnwlfmjac preimage=d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee
t=2001-09-09T01:51:10+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
t=2001-09-09T01:51:20+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=00f971921465de77c458075e3df560f7c52b2168d0ec3f253406dc8049e772ab address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
t=2001-09-09T01:51:30+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:00f971921465de77c458075e3df560f7c52b2168d0ec3f253406dc8049e772ab TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[49 177 249 153 224 154 117 249 195 203 229 70 130 23 234 161 107 75 142 202 190 243 184 236 128 248 61 216 27 70 255 253]}" preimage=d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee
`)
	activateHtlc(z)

	preimage := preimageZ
	lock := crypto.Hash(preimage)
	lockQ := crypto.Hash(preimageQ)

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

	expectedId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

	// user 1 creates another htlc for user 2
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User1.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.CreateHtlcMethodName,
			g.User2.Address,                // hashlocked
			int64(genesisTimestamp+300),    // expiration time
			uint8(definition.HashTypeSHA3), // hash type
			uint8(32),                      // max preimage size
			lockQ,                          // hashlock
		),
		TokenStandard: types.QsrTokenStandard,
		Amount:        big.NewInt(10 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	expectedId2 := types.HexToHashPanic("00f971921465de77c458075e3df560f7c52b2168d0ec3f253406dc8049e772ab")

	// user 2 disables proxy unlock
	defer z.CallContract(&nom.AccountBlock{
		Address:       g.User2.Address,
		ToAddress:     types.HtlcContract,
		Data:          definition.ABIHtlc.PackMethodPanic(definition.DenyHtlcProxyUnlockMethodName),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcProxyUnlockStatus(g.User2.Address)).Equals(t, `false`)

	// user 2 allows proxy unlock
	defer z.CallContract(&nom.AccountBlock{
		Address:       g.User2.Address,
		ToAddress:     types.HtlcContract,
		Data:          definition.ABIHtlc.PackMethodPanic(definition.AllowHtlcProxyUnlockMethodName),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcProxyUnlockStatus(g.User2.Address)).Equals(t, `true`)

	// user 3 tries to unlock the first htlc with wrong preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimageQ,  // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, constants.ErrInvalidPreimage)
	z.InsertNewMomentum()

	// user 3 unlocks with the first htlc with the correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User3.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId, // entry id
			preimage,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()

	// user 2 tries to unlock the second htlc with wrong preimage
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

	// user 2 unlocks with the second htlc with the correct preimage
	defer z.CallContract(&nom.AccountBlock{
		Address:   g.User2.Address,
		ToAddress: types.HtlcContract,
		Data: definition.ABIHtlc.PackMethodPanic(definition.UnlockHtlcMethodName,
			expectedId2, // entry id
			preimageQ,   // preimage
		),
		TokenStandard: types.ZnnTokenStandard,
		Amount:        big.NewInt(0 * g.Zexp),
	}).Error(t, nil)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Error(t, constants.ErrDataNonExistent)

	autoreceive(t, z, g.User2.Address)
	z.InsertNewMomentum()
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-10)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, (120000-10)*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, (8000+10)*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, (80000+10)*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)
}

func TestHtlc_reclaim_access(t *testing.T) {
	z := mock.NewMockZenon(t)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:10+0000 lvl=dbug msg="invalid reclaim - permission denied" module=embedded contract=htlc id=dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx
t=2001-09-09T01:50:20+0000 lvl=dbug msg="invalid reclaim - permission denied" module=embedded contract=htlc id=dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 address=z1qrs2lpccnsneglhnnfwvlsj0qncnxjnwlfmjac
t=2001-09-09T01:53:20+0000 lvl=dbug msg="invalid reclaim - permission denied" module=embedded contract=htlc id=dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx
t=2001-09-09T01:53:30+0000 lvl=dbug msg="invalid reclaim - permission denied" module=embedded contract=htlc id=dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 address=z1qrs2lpccnsneglhnnfwvlsj0qncnxjnwlfmjac
t=2001-09-09T01:53:40+0000 lvl=dbug msg=reclaimed module=embedded contract=htlc htlcInfo="&{Id:dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1qsrxxxxxxxxxxxxxmrhjll Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId := types.HexToHashPanic("dbc7d894a9acd06ac2017301c1b8c5ac017327095d0af5062cb902b8077cbdc1")

	// user 2 tries to reclaim unexpired
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

	// user 3 tries to reclaim unexpired
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

	// expire the entry
	z.InsertMomentumsTo(40)

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
}

func TestHtlc_nonexistent(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg="invalid unlock - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
t=2001-09-09T01:50:10+0000 lvl=dbug msg="invalid reclaim - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
`)
	activateHtlc(z)

	preimage := preimageZ
	nonexistentId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

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
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
t=2001-09-09T01:50:40+0000 lvl=dbug msg="invalid unlock - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
t=2001-09-09T01:50:50+0000 lvl=dbug msg="invalid reclaim - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
`)
	activateHtlc(z)

	preimage := preimageZ
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

	htlcId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

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
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:53:20+0000 lvl=dbug msg=reclaimed module=embedded contract=htlc htlcInfo="&{Id:279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:53:40+0000 lvl=dbug msg="invalid unlock - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
t=2001-09-09T01:53:50+0000 lvl=dbug msg="invalid reclaim - entry does not exist" module=embedded contract=htlc id=279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5 address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz
`)
	activateHtlc(z)

	preimage := preimageZ
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

	htlcId := types.HexToHashPanic("279cadb7e128de79af66d1f4abfe819350f7245e5d7036e16165f2e7ecf4bde5")

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
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg="invalid create - cannot create already expired" module=embedded contract=htlc address=z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz time=1000000200 expiration-time=999999700
`)
	activateHtlc(z)

	preimage := preimageZ
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
}

// test unlock htlc with ErrInvalidPreimage
func TestHtlc_unlock_long_preimage(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:5eab16285e906726cfc11419f29146c6ee765b8b2c044c2b60b33ddff91925ed TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[8 20 65 196 102 82 153 51 204 41 55 249 51 226 36 239 65 178 93 135 130 66 232 145 62 36 203 88 30 225 243 37]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg="invalid unlock - preimage size greater than entry KeyMaxSize" module=embedded contract=htlc id=5eab16285e906726cfc11419f29146c6ee765b8b2c044c2b60b33ddff91925ed address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage-size=64 max-size=32
`)
	activateHtlc(z)

	// ideally for this test we would have a known hash collision with 2 preimages of different sizes
	// this test relies on knowledge that if the preimage produces the right hash, that it won't skip the size check
	preimage := preimageZlong
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

	expectedId := types.HexToHashPanic("5eab16285e906726cfc11419f29146c6ee765b8b2c044c2b60b33ddff91925ed")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "5eab16285e906726cfc11419f29146c6ee765b8b2c044c2b60b33ddff91925ed",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "CBRBxGZSmTPMKTf5M+Ik70GyXYeCQuiRPiTLWB7h8yU="
}
`)

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
	z.InsertNewMomentum()

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-10)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 10*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

}

// SHA256 testing
// blackbox testing principles would dictate that we run the same set of tests
// as above with each different hash type
// But until we have parameterized tests, will add streamlined higher value tests
// happy path and making sure sha3 preimage can't unlock sha256 hashlock
// and vice versa

func TestHtlc_unlockSHA256(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:04b4c0870fc824a8a68917b862a4cbf19c66a2b8091bcfbf31d4459aff757dd7 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:1 KeyMaxSize:32 HashLock:[205 134 140 113 161 201 215 44 84 153 182 139 176 110 237 55 66 119 227 51 109 132 58 15 17 145 68 97 195 93 208 54]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=04b4c0870fc824a8a68917b862a4cbf19c66a2b8091bcfbf31d4459aff757dd7 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage=d70b59367334f9c6d4771059093ec11cb505d7b2b0e233cc8bde00fe7aec3cee
t=2001-09-09T01:50:30+0000 lvl=dbug msg=unlocked module=embedded contract=htlc htlcInfo="&{Id:04b4c0870fc824a8a68917b862a4cbf19c66a2b8091bcfbf31d4459aff757dd7 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:1 KeyMaxSize:32 HashLock:[205 134 140 113 161 201 215 44 84 153 182 139 176 110 237 55 66 119 227 51 109 132 58 15 17 145 68 97 195 93 208 54]}" preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId := types.HexToHashPanic("04b4c0870fc824a8a68917b862a4cbf19c66a2b8091bcfbf31d4459aff757dd7")

	common.Json(htlcApi.GetHtlcInfoById(expectedId)).Equals(t, `
{
	"id": "04b4c0870fc824a8a68917b862a4cbf19c66a2b8091bcfbf31d4459aff757dd7",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 1,
	"keyMaxSize": 32,
	"hashLock": "zYaMcaHJ1yxUmbaLsG7tN0J34zNthDoPEZFEYcNd0DY="
}
`)

	// user 2 tries to unlock with wrong preimage
	wrong_preimage := preimageQ
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

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-10)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, (8000+10)*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 0*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

}

func TestHtlc_hashType(t *testing.T) {
	z := mock.NewMockZenon(t)
	htlcApi := embedded.NewHtlcApi(z)
	defer z.StopPanic()
	defer z.SaveLogs(common.EmbeddedLogger).Equals(t, `
t=2001-09-09T01:46:50+0000 lvl=dbug msg=created module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:false EnforcementHeight:0}"
t=2001-09-09T01:47:00+0000 lvl=dbug msg=activated module=embedded contract=spork spork="&{Id:664147f0c0a127bb4388bf8ff9a2ce777c9cc5ce9f04f9a6d418a32ef3f481c9 Name:spork-htlc Description:activate spork for htlc Activated:true EnforcementHeight:9}"
t=2001-09-09T01:50:00+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:c9fb086e859974b9fa8a0b9ebd7bffe671b8cff3de8cc9787fab73b0cd41cdaf TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:0 KeyMaxSize:32 HashLock:[205 134 140 113 161 201 215 44 84 153 182 139 176 110 237 55 66 119 227 51 109 132 58 15 17 145 68 97 195 93 208 54]}"
t=2001-09-09T01:50:20+0000 lvl=dbug msg=created module=embedded contract=htlc htlcInfo="{Id:a21f1b6b7057f748d59ac110d2d8395675225945ae5250659696a65004f0dda9 TimeLocked:z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz HashLocked:z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx TokenStandard:zts1znnxxxxxxxxxxxxx9z4ulx Amount:+1000000000 ExpirationTime:1000000300 HashType:1 KeyMaxSize:32 HashLock:[21 222 16 14 131 114 144 54 199 167 125 27 96 136 33 66 98 184 56 231 39 1 59 6 138 82 220 166 222 228 87 203]}"
t=2001-09-09T01:50:40+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=c9fb086e859974b9fa8a0b9ebd7bffe671b8cff3de8cc9787fab73b0cd41cdaf address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
t=2001-09-09T01:50:50+0000 lvl=dbug msg="invalid unlock - wrong preimage" module=embedded contract=htlc id=a21f1b6b7057f748d59ac110d2d8395675225945ae5250659696a65004f0dda9 address=z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx preimage=b7845adcd41eec4e4fa1cc75a868014811b575942c6e4a72551bc01f63705634
`)
	activateHtlc(z)

	preimage := preimageZ
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

	expectedId1 := types.HexToHashPanic("c9fb086e859974b9fa8a0b9ebd7bffe671b8cff3de8cc9787fab73b0cd41cdaf")
	common.Json(htlcApi.GetHtlcInfoById(expectedId1)).Equals(t, `
{
	"id": "c9fb086e859974b9fa8a0b9ebd7bffe671b8cff3de8cc9787fab73b0cd41cdaf",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 0,
	"keyMaxSize": 32,
	"hashLock": "zYaMcaHJ1yxUmbaLsG7tN0J34zNthDoPEZFEYcNd0DY="
}
`)

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

	expectedId2 := types.HexToHashPanic("a21f1b6b7057f748d59ac110d2d8395675225945ae5250659696a65004f0dda9")
	common.Json(htlcApi.GetHtlcInfoById(expectedId2)).Equals(t, `
{
	"id": "a21f1b6b7057f748d59ac110d2d8395675225945ae5250659696a65004f0dda9",
	"timeLocked": "z1qzal6c5s9rjnnxd2z7dvdhjxpmmj4fmw56a0mz",
	"hashLocked": "z1qr4pexnnfaexqqz8nscjjcsajy5hdqfkgadvwx",
	"tokenStandard": "zts1znnxxxxxxxxxxxxx9z4ulx",
	"amount": 1000000000,
	"expirationTime": 1000000300,
	"hashType": 1,
	"keyMaxSize": 32,
	"hashLock": "Fd4QDoNykDbHp30bYIghQmK4OOcnATsGilLcpt7kV8s="
}
`)

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

	z.ExpectBalance(g.User1.Address, types.ZnnTokenStandard, (12000-20)*g.Zexp)
	z.ExpectBalance(g.User1.Address, types.QsrTokenStandard, 120000*g.Zexp)

	z.ExpectBalance(g.User2.Address, types.ZnnTokenStandard, 8000*g.Zexp)
	z.ExpectBalance(g.User2.Address, types.QsrTokenStandard, 80000*g.Zexp)

	z.ExpectBalance(types.HtlcContract, types.ZnnTokenStandard, 20*g.Zexp)
	z.ExpectBalance(types.HtlcContract, types.QsrTokenStandard, 0*g.Zexp)

}
