// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/luxfi/consensus/core/choices"
	"github.com/luxfi/consensus/engine/dag/vertex"
	"github.com/luxfi/ids"
)

var _ vertex.DAGVM = (*VM)(nil)

// FeedRoundKey is the conflict key for the Oracle VM: (feedID, round/epoch).
// Same feed+round conflicts; different feeds commute.
type FeedRoundKey struct {
	FeedID ids.ID
	Round  uint64
}

// OracleVertex represents a DAG vertex in the Oracle chain.
type OracleVertex struct {
	id      ids.ID
	bytes   []byte
	height  uint64
	epoch   uint32
	parents []ids.ID
	txIDs   []ids.ID
	status  choices.Status

	observations []*Observation
	aggregations []*AggregatedValue
	feedUpdates  []*Feed
	keys         []FeedRoundKey
	vm           *VM
}

func (v *OracleVertex) ID() ids.ID          { return v.id }
func (v *OracleVertex) Bytes() []byte        { return v.bytes }
func (v *OracleVertex) Height() uint64       { return v.height }
func (v *OracleVertex) Epoch() uint32        { return v.epoch }
func (v *OracleVertex) Parents() []ids.ID    { return v.parents }
func (v *OracleVertex) Txs() []ids.ID        { return v.txIDs }
func (v *OracleVertex) Status() choices.Status { return v.status }

func (v *OracleVertex) Verify(ctx context.Context) error {
	for _, obs := range v.observations {
		if _, exists := v.vm.feeds[obs.FeedID]; !exists {
			return ErrFeedNotFound
		}
	}
	return nil
}

func (v *OracleVertex) Accept(ctx context.Context) error {
	v.status = choices.Accepted

	v.vm.mu.Lock()
	defer v.vm.mu.Unlock()

	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := v.vm.db.Put(v.id[:], b); err != nil {
		return err
	}
	v.vm.lastAcceptedID = v.id
	delete(v.vm.pendingBlocks, v.id)
	return nil
}

func (v *OracleVertex) Reject(ctx context.Context) error {
	v.status = choices.Rejected
	v.vm.mu.Lock()
	delete(v.vm.pendingBlocks, v.id)
	v.vm.mu.Unlock()
	return nil
}

// conflictKeySet returns the set of FeedRoundKeys for conflict detection.
func (v *OracleVertex) conflictKeySet() map[FeedRoundKey]struct{} {
	s := make(map[FeedRoundKey]struct{}, len(v.keys))
	for _, k := range v.keys {
		s[k] = struct{}{}
	}
	return s
}

// Conflicts returns true if this vertex and other share any (feedID, round) pair.
func (v *OracleVertex) Conflicts(other *OracleVertex) bool {
	ours := v.conflictKeySet()
	for _, k := range other.keys {
		if _, ok := ours[k]; ok {
			return true
		}
	}
	return false
}

// ConflictsVertex performs the same check against the vertex.Vertex interface.
func (v *OracleVertex) ConflictsVertex(other vertex.Vertex) bool {
	ov, ok := other.(*OracleVertex)
	if !ok {
		return false
	}
	return v.Conflicts(ov)
}

// extractFeedRoundKeys derives conflict keys from observations and aggregations.
func extractFeedRoundKeys(obs []*Observation, aggs []*AggregatedValue) []FeedRoundKey {
	seen := make(map[FeedRoundKey]struct{})
	var keys []FeedRoundKey
	for _, o := range obs {
		k := FeedRoundKey{FeedID: o.FeedID, Round: uint64(o.Timestamp.UnixMilli())}
		if _, dup := seen[k]; !dup {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	for _, a := range aggs {
		k := FeedRoundKey{FeedID: a.FeedID, Round: a.Epoch}
		if _, dup := seen[k]; !dup {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	return keys
}

func (v *OracleVertex) computeID() ids.ID {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, v.height)
	binary.Write(h, binary.BigEndian, v.epoch)
	for _, p := range v.parents {
		h.Write(p[:])
	}
	for _, k := range v.keys {
		h.Write(k.FeedID[:])
		binary.Write(h, binary.BigEndian, k.Round)
	}
	return ids.ID(h.Sum(nil))
}

// BuildVertex creates a vertex from pending observations and aggregations.
func (vm *VM) BuildVertex(ctx context.Context) (vertex.Vertex, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return nil, ErrNotInitialized
	}

	parent := vm.lastAccepted
	if parent == nil {
		return nil, errors.New("no parent block")
	}

	// Collect all pending observations
	var allObs []*Observation
	for _, obs := range vm.pendingObs {
		allObs = append(allObs, obs...)
	}

	if len(allObs) == 0 {
		return nil, errors.New("no pending observations")
	}

	// Batch observations by unique (feedID, round)
	seen := make(map[FeedRoundKey]struct{})
	var batch []*Observation
	for _, obs := range allObs {
		k := FeedRoundKey{FeedID: obs.FeedID, Round: uint64(obs.Timestamp.UnixMilli())}
		seen[k] = struct{}{}
		batch = append(batch, obs)
	}

	keys := extractFeedRoundKeys(batch, nil)
	txIDs := make([]ids.ID, len(batch))
	for i, obs := range batch {
		h := sha256.New()
		h.Write(obs.FeedID[:])
		h.Write(obs.Value)
		h.Write(obs.OperatorID[:])
		txIDs[i] = ids.ID(h.Sum(nil))
	}

	v := &OracleVertex{
		height:       parent.Height_ + 1,
		epoch:        0,
		parents:      []ids.ID{vm.lastAcceptedID},
		txIDs:        txIDs,
		observations: batch,
		keys:         keys,
		status:       choices.Processing,
		vm:           vm,
	}
	v.id = v.computeID()
	v.bytes, _ = json.Marshal(v)

	// Clear consumed pending observations
	vm.pendingObs = make(map[ids.ID][]*Observation)

	return v, nil
}

// ParseVertex deserializes a vertex from bytes.
func (vm *VM) ParseVertex(ctx context.Context, b []byte) (vertex.Vertex, error) {
	v := &OracleVertex{vm: vm}
	if err := json.Unmarshal(b, v); err != nil {
		return nil, err
	}
	v.keys = extractFeedRoundKeys(v.observations, v.aggregations)
	v.id = v.computeID()
	v.bytes = b
	return v, nil
}
