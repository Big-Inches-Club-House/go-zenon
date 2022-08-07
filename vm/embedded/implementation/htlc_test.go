package implementation

import (
	"testing"

	g "github.com/zenon-network/go-zenon/chain/genesis/mock"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/vm/constants"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
)

var (
	defaultHtlc = definition.CreateHtlcParam{
		HashLocked:     g.User1.Address,
		ExpirationTime: 1000000000,
		HashType:       0,
		KeyMaxSize:     32,
		HashLock:       []byte("pxjuP+c5vWQ18L17tO6Q4d7/I0M3LZKgRZLCajm1cKg="),
	}
)

func TestHtlc_HashType(t *testing.T) {
	htlc := defaultHtlc
	htlc.HashType = 1
	common.ExpectError(t, checkHtlc(htlc), constants.ErrInvalidHashType)
}

func TestHtlc_LockLength(t *testing.T) {
	htlc := defaultHtlc
	htlc.HashLock = htlc.HashLock[1:]
	common.ExpectError(t, checkHtlc(htlc), constants.ErrInvalidHashDigest)
}
