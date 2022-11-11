package implementation

import (
	"encoding/base64"
	"testing"

	g "github.com/zenon-network/go-zenon/chain/genesis/mock"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
)

var (
	// i completely forget what the preimage for this is lol, some meme probably
	// encoding/json marshals []byte as base64 following RFC 4648
	hashlock, _ = base64.StdEncoding.DecodeString("pxjuP+c5vWQ18L17tO6Q4d7/I0M3LZKgRZLCajm1cKg=")
	defaultHtlc = definition.CreateHtlcParam{
		HashLocked:     g.User1.Address,
		ExpirationTime: 1000000000,
		HashType:       0,
		KeyMaxSize:     32,
		HashLock:       hashlock,
	}
)

func TestHtlc_HashType(t *testing.T) {
	htlc := defaultHtlc
	common.ExpectError(t, checkHtlc(htlc), nil)
	htlc.HashType = 1
	common.ExpectError(t, checkHtlc(htlc), nil)
	htlc.HashType = 2
	common.ExpectError(t, checkHtlc(htlc), constants.ErrInvalidHashType)
}

func TestHtlc_LockLength(t *testing.T) {
	htlc := defaultHtlc
	htlc.HashLock = htlc.HashLock[1:]
	common.ExpectError(t, checkHtlc(htlc), constants.ErrInvalidHashDigest)
	htlc.HashType = 1
	common.ExpectError(t, checkHtlc(htlc), constants.ErrInvalidHashDigest)
}
