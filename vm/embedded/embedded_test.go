package embedded

import (
	"encoding/hex"
	"fmt"
	"sort"
	"testing"

	"github.com/zenon-network/go-zenon/common"
)

func TestDumpContractsABIMethods(t *testing.T) {
	dumps := make([]string, 0)
	// TODO update/add tests for AZ and HTLC
	for addr, contract := range originEmbedded {
		// this test is fundamentally broken and failing currently
		// gets the contracts list from originEmbedded
		// but dumps all methods regardless of when it was activated
		// need to loop over embeddedImplementation.m
		for _, method := range contract.abi.Methods {
			dumps = append(dumps, fmt.Sprintf(`{"address":"%v", "name":"%v", "id":"%v", "signature":"%v"}`, addr, method.Name, hex.EncodeToString(method.Id()), method.Sig()))
		}
	}
	sort.Strings(dumps)
	dump := "[\n"
	for i := range dumps {
		if i+1 != len(dumps) {
			dump = dump + dumps[i] + "\n"
		} else {
			dump = dump + dumps[i] + "\n"
		}
	}
	dump += "]\n"

	common.Expect(t, dump, `
[
{"address":"z1qxemdeddedxaccelerat0rxxxxxxxxxxp4tk22", "name":"Donate", "id":"cb7f8b2a", "signature":"Donate()"}
{"address":"z1qxemdeddedxlyquydytyxxxxxxxxxxxxflaaae", "name":"Donate", "id":"cb7f8b2a", "signature":"Donate()"}
{"address":"z1qxemdeddedxlyquydytyxxxxxxxxxxxxflaaae", "name":"Update", "id":"20093ea6", "signature":"Update()"}
{"address":"z1qxemdeddedxplasmaxxxxxxxxxxxxxxxxsctrp", "name":"CancelFuse", "id":"f9ca9dc3", "signature":"CancelFuse(hash)"}
{"address":"z1qxemdeddedxplasmaxxxxxxxxxxxxxxxxsctrp", "name":"Fuse", "id":"5ac942e8", "signature":"Fuse(address)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"CollectReward", "id":"af43d3f0", "signature":"CollectReward()"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"Delegate", "id":"7c2d5d6e", "signature":"Delegate(string)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"DepositQsr", "id":"d49577f4", "signature":"DepositQsr()"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"Register", "id":"644de927", "signature":"Register(string,address,address,uint8,uint8)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"RegisterLegacy", "id":"e4588207", "signature":"RegisterLegacy(string,address,address,uint8,uint8,string,string)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"Revoke", "id":"95631306", "signature":"Revoke(string)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"Undelegate", "id":"7e8952c8", "signature":"Undelegate()"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"Update", "id":"20093ea6", "signature":"Update()"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"UpdatePillar", "id":"de0ae34b", "signature":"UpdatePillar(string,address,address,uint8,uint8)"}
{"address":"z1qxemdeddedxpyllarxxxxxxxxxxxxxxxsy3fmg", "name":"WithdrawQsr", "id":"b3d658fd", "signature":"WithdrawQsr()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"CollectReward", "id":"af43d3f0", "signature":"CollectReward()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"DepositQsr", "id":"d49577f4", "signature":"DepositQsr()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"Register", "id":"4dd23517", "signature":"Register()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"Revoke", "id":"58363e24", "signature":"Revoke()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"Update", "id":"20093ea6", "signature":"Update()"}
{"address":"z1qxemdeddedxsentynelxxxxxxxxxxxxxwy0r2r", "name":"WithdrawQsr", "id":"b3d658fd", "signature":"WithdrawQsr()"}
{"address":"z1qxemdeddedxsp0rkxxxxxxxxxxxxxxxx956u48", "name":"ActivateSpork", "id":"25c54e96", "signature":"ActivateSpork(hash)"}
{"address":"z1qxemdeddedxsp0rkxxxxxxxxxxxxxxxx956u48", "name":"CreateSpork", "id":"b602e311", "signature":"CreateSpork(string,string)"}
{"address":"z1qxemdeddedxstakexxxxxxxxxxxxxxxxjv8v62", "name":"Cancel", "id":"5a92fe32", "signature":"Cancel(hash)"}
{"address":"z1qxemdeddedxstakexxxxxxxxxxxxxxxxjv8v62", "name":"CollectReward", "id":"af43d3f0", "signature":"CollectReward()"}
{"address":"z1qxemdeddedxstakexxxxxxxxxxxxxxxxjv8v62", "name":"Stake", "id":"d802845a", "signature":"Stake(int64)"}
{"address":"z1qxemdeddedxstakexxxxxxxxxxxxxxxxjv8v62", "name":"Update", "id":"20093ea6", "signature":"Update()"}
{"address":"z1qxemdeddedxswapxxxxxxxxxxxxxxxxxxl4yww", "name":"RetrieveAssets", "id":"47f12c81", "signature":"RetrieveAssets(string,string)"}
{"address":"z1qxemdeddedxt0kenxxxxxxxxxxxxxxxxh9amk0", "name":"Burn", "id":"3395ab94", "signature":"Burn()"}
{"address":"z1qxemdeddedxt0kenxxxxxxxxxxxxxxxxh9amk0", "name":"IssueToken", "id":"bc410b91", "signature":"IssueToken(string,string,string,uint256,uint256,uint8,bool,bool,bool)"}
{"address":"z1qxemdeddedxt0kenxxxxxxxxxxxxxxxxh9amk0", "name":"Mint", "id":"cd70f9bc", "signature":"Mint(tokenStandard,uint256,address)"}
{"address":"z1qxemdeddedxt0kenxxxxxxxxxxxxxxxxh9amk0", "name":"UpdateToken", "id":"2a3cf32c", "signature":"UpdateToken(tokenStandard,address,bool,bool)"}
]`)
}
