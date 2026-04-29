// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/node/vms"
)

var _ vms.Factory = (*Factory)(nil)

// VMID is the unique identifier for OracleVM (O-Chain)
var VMID = ids.ID{'o', 'r', 'a', 'c', 'l', 'e', 'v', 'm'}

// Factory creates new OracleVM instances
type Factory struct{}

// New returns a new instance of the OracleVM
func (f *Factory) New(logger log.Logger) (interface{}, error) {
	return &VM{
		feeds:         make(map[ids.ID]*Feed),
		pendingObs:    make(map[ids.ID][]*Observation),
		values:        make(map[ids.ID]map[uint64]*AggregatedValue),
		pendingBlocks: make(map[ids.ID]*Block),
	}, nil
}
