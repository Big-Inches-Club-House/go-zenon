package vm_context

import (
	"github.com/zenon-network/go-zenon/common"
	"github.com/zenon-network/go-zenon/common/types"
)

func (ctx *accountVmContext) IsAcceleratorSporkEnforced() bool {
	active, err := ctx.momentumStore.IsSporkActive(types.AcceleratorSpork)
	common.DealWithErr(err)
	return active
}

func (ctx *accountVmContext) IsHtlcSporkEnforced() bool {
	active, err := ctx.momentumStore.IsSporkActive(types.HtlcSpork)
	common.DealWithErr(err)
	return active
}
