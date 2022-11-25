package embedded

import (
	"sort"

	"github.com/inconshreveable/log15"

	"github.com/zenon-network/go-zenon/chain"
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/types"
	"github.com/zenon-network/go-zenon/consensus"
	"github.com/zenon-network/go-zenon/rpc/api"
	"github.com/zenon-network/go-zenon/vm/embedded/definition"
	"github.com/zenon-network/go-zenon/zenon"
)

type HtlcApi struct {
	chain chain.Chain
	z     zenon.Zenon
	cs    consensus.Consensus
	log   log15.Logger
}

func NewHtlcApi(z zenon.Zenon) *HtlcApi {
	return &HtlcApi{
		chain: z.Chain(),
		z:     z,
		cs:    z.Consensus(),
		log:   common.RPCLogger.New("module", "embedded_htlc_api"),
	}
}

type SortHtlcInfoByExpiration []*definition.HtlcInfo

func (a SortHtlcInfoByExpiration) Len() int      { return len(a) }
func (a SortHtlcInfoByExpiration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortHtlcInfoByExpiration) Less(i, j int) bool {
	if a[i].ExpirationTime == a[j].ExpirationTime {
		return a[i].HashLocked.String() < a[j].HashLocked.String()
	}
	return a[i].ExpirationTime < a[j].ExpirationTime
}

func (a *HtlcApi) GetHtlcInfoById(id types.Hash) (*definition.HtlcInfo, error) {

	_, context, err := api.GetFrontierContext(a.chain, types.HtlcContract)
	if err != nil {
		return nil, err
	}

	htlcInfo, err := definition.GetHtlcInfo(context.Storage(), id)
	if err != nil {
		return nil, err
	}

	return htlcInfo, nil
}

type HtlcInfoList struct {
	Count uint32                 `json:"count"`
	List  []*definition.HtlcInfo `json:"list"`
}

func (a *HtlcApi) GetHtlcInfosByLockTypeAddress(locktype []byte, address types.Address, pageIndex, pageSize uint32) (*HtlcInfoList, error) {
	if pageSize > api.RpcMaxPageSize {
		return nil, api.ErrPageSizeParamTooBig
	}

	_, context, err := api.GetFrontierContext(a.chain, types.HtlcContract)
	if err != nil {
		return nil, err
	}

	refs, err := definition.GetHtlcRefList(context.Storage(), locktype, address)
	if err != nil {
		return nil, err
	}

	list := make([]*definition.HtlcInfo, 0)
	for _, r := range refs {
		l, err := definition.GetHtlcInfo(context.Storage(), r.Id)
		if err != nil {
			return nil, err
		}
		list = append(list, l)
	}

	sort.Sort(SortHtlcInfoByExpiration(list))
	listLen := len(list)
	start, end := api.GetRange(pageIndex, pageSize, uint32(listLen))

	return &HtlcInfoList{
		Count: uint32(listLen),
		List:  list[start:end],
	}, nil
}

func (a *HtlcApi) GetHtlcInfosByTimeLockedAddress(address types.Address, pageIndex, pageSize uint32) (*HtlcInfoList, error) {
	return a.GetHtlcInfosByLockTypeAddress([]byte{2}, address, pageIndex, pageSize)
}

func (a *HtlcApi) GetHtlcInfosByHashLockedAddress(address types.Address, pageIndex, pageSize uint32) (*HtlcInfoList, error) {
	return a.GetHtlcInfosByLockTypeAddress([]byte{3}, address, pageIndex, pageSize)
}
