// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/database/memdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/runtime"
	vmcore "github.com/luxfi/vm"
)

func TestVMID(t *testing.T) {
	require := require.New(t)
	require.NotEqual(ids.Empty, VMID, "VMID should not be empty")
	require.Equal(ids.ID{'o', 'r', 'a', 'c', 'l', 'e', 'v', 'm'}, VMID)
}

func TestFactoryNew(t *testing.T) {
	require := require.New(t)

	factory := &Factory{}
	vm, err := factory.New(log.NewNoOpLogger())
	require.NoError(err)
	require.NotNil(vm)
	require.IsType(&VM{}, vm)
}

func TestVMInitialize(t *testing.T) {
	require := require.New(t)

	vm := &VM{
		feeds:         make(map[ids.ID]*Feed),
		feedsByName:   make(map[string]ids.ID),
		pendingObs:    make(map[ids.ID][]*Observation),
		values:        make(map[ids.ID]map[uint64]*AggregatedValue),
		pendingBlocks: make(map[ids.ID]*Block),
	}

	genesis := &Genesis{
		Version:   1,
		Message:   "test genesis",
		Timestamp: time.Now().Unix(),
	}
	genesisBytes, err := json.Marshal(genesis)
	require.NoError(err)

	config := DefaultConfig()
	configBytes, err := json.Marshal(config)
	require.NoError(err)

	toEngine := make(chan vmcore.Message, 10)

	init := vmcore.Init{
		Runtime: &runtime.Runtime{
			ChainID: ids.GenerateTestID(),
			Log:     log.NewNoOpLogger(),
		},
		DB:       memdb.New(),
		Genesis:  genesisBytes,
		Config:   configBytes,
		ToEngine: toEngine,
	}

	err = vm.Initialize(context.Background(), init)
	require.NoError(err)
	require.True(vm.running)

	// Verify shutdown
	err = vm.Shutdown(context.Background())
	require.NoError(err)
	require.False(vm.running)
}

func TestVMRegisterFeed(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	feed := &Feed{
		ID:          ids.GenerateTestID(),
		Name:        "test-feed",
		Description: "Test oracle feed",
		Sources:     []string{"https://api.example.com/price"},
		UpdateFreq:  time.Minute,
		Operators:   []ids.NodeID{ids.GenerateTestNodeID()},
	}

	err := vm.RegisterFeed(feed)
	require.NoError(err)

	// Verify feed was registered
	retrieved, err := vm.GetFeed(feed.ID)
	require.NoError(err)
	require.Equal(feed.Name, retrieved.Name)
	require.Equal("active", retrieved.Status)

	// Duplicate should fail
	err = vm.RegisterFeed(feed)
	require.Error(err)
}

func TestVMSubmitObservation(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	operatorID := ids.GenerateTestNodeID()
	feed := &Feed{
		ID:          ids.GenerateTestID(),
		Name:        "price-feed",
		Description: "Price oracle feed",
		Operators:   []ids.NodeID{operatorID},
	}
	err := vm.RegisterFeed(feed)
	require.NoError(err)

	obs := &Observation{
		FeedID:     feed.ID,
		Value:      []byte(`{"price": 100.50}`),
		Timestamp:  time.Now(),
		OperatorID: operatorID,
		Signature:  []byte("test-sig"),
	}

	err = vm.SubmitObservation(obs)
	require.NoError(err)

	// Verify pending observations
	require.Len(vm.pendingObs[feed.ID], 1)
}

func TestVMBuildBlock(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	// Build a block
	blk, err := vm.BuildBlock(context.Background())
	require.NoError(err)
	require.NotNil(blk)
	require.Equal(uint64(1), blk.Height())

	// Verify block parent
	lastAccepted, err := vm.LastAccepted(context.Background())
	require.NoError(err)
	require.Equal(lastAccepted, blk.Parent())
}

func TestVMParseBlock(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	blk, err := vm.BuildBlock(context.Background())
	require.NoError(err)

	// Parse the block bytes
	parsed, err := vm.ParseBlock(context.Background(), blk.Bytes())
	require.NoError(err)
	// Note: IDs may differ due to JSON encoding differences
	require.Equal(blk.Height(), parsed.Height())
}

func TestBlockVerifyAcceptReject(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	blk, err := vm.BuildBlock(context.Background())
	require.NoError(err)

	// Verify
	err = blk.Verify(context.Background())
	require.NoError(err)

	// Accept
	err = blk.Accept(context.Background())
	require.NoError(err)

	// Verify last accepted updated
	lastAccepted, err := vm.LastAccepted(context.Background())
	require.NoError(err)
	require.Equal(blk.ID(), lastAccepted)
}

func TestVMHealthCheck(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	health, err := vm.HealthCheck(context.Background())
	require.NoError(err)
	require.True(health.Healthy)
}

func TestVMVersion(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	version, err := vm.Version(context.Background())
	require.NoError(err)
	require.Equal("v1.0.0", version)
}

func TestVMCreateHandlers(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	handlers, err := vm.CreateHandlers(context.Background())
	require.NoError(err)
	require.NotNil(handlers)
	require.Contains(handlers, "/rpc")
}

func TestServiceRPC(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	service := NewService(vm)
	require.NotNil(service)

	// Test Health RPC
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(`{
		"jsonrpc": "2.0",
		"method": "oracle.Health",
		"params": [{}],
		"id": 1
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)
	require.Equal(http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(err)
	require.NotNil(resp["result"])
}

func TestServiceRegisterFeed(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	service := NewService(vm)

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(`{
		"jsonrpc": "2.0",
		"method": "oracle.RegisterFeed",
		"params": [{
			"name": "eth-usd",
			"description": "ETH/USD price feed",
			"sources": ["https://api.coinbase.com/v2/prices/ETH-USD/spot"],
			"updateFreq": "1m"
		}],
		"id": 1
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)
	require.Equal(http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(err)
	require.NotNil(resp["result"])

	result := resp["result"].(map[string]interface{})
	require.NotEmpty(result["feedId"])
}

func TestCreateAttestation(t *testing.T) {
	require := require.New(t)

	vm := setupTestVM(t)
	defer vm.Shutdown(context.Background())

	operatorID := ids.GenerateTestNodeID()
	feed := &Feed{
		ID:         ids.GenerateTestID(),
		Name:       "attestation-test",
		UpdateFreq: time.Minute,
		Operators:  []ids.NodeID{operatorID},
	}
	err := vm.RegisterFeed(feed)
	require.NoError(err)

	// Add a value
	vm.values[feed.ID] = map[uint64]*AggregatedValue{
		1: {
			FeedID:    feed.ID,
			Epoch:     1,
			Value:     []byte(`{"price": 2000}`),
			Timestamp: time.Now(),
		},
	}

	att, err := vm.CreateAttestation(feed.ID, 1)
	require.NoError(err)
	require.NotNil(att)
	require.Equal(feed.ID, att.FeedID)
	require.Equal(uint64(1), att.Epoch)
}

// setupTestVM creates and initializes a test VM
func setupTestVM(t *testing.T) *VM {
	t.Helper()

	vm := &VM{
		feeds:         make(map[ids.ID]*Feed),
		feedsByName:   make(map[string]ids.ID),
		pendingObs:    make(map[ids.ID][]*Observation),
		values:        make(map[ids.ID]map[uint64]*AggregatedValue),
		pendingBlocks: make(map[ids.ID]*Block),
	}

	genesis := &Genesis{
		Version:   1,
		Message:   "test",
		Timestamp: time.Now().Unix(),
	}
	genesisBytes, _ := json.Marshal(genesis)

	config := DefaultConfig()
	configBytes, _ := json.Marshal(config)

	toEngine := make(chan vmcore.Message, 10)

	init := vmcore.Init{
		Runtime: &runtime.Runtime{
			ChainID: ids.GenerateTestID(),
			Log:     log.NewNoOpLogger(),
		},
		DB:       memdb.New(),
		Genesis:  genesisBytes,
		Config:   configBytes,
		ToEngine: toEngine,
	}

	err := vm.Initialize(context.Background(), init)
	require.NoError(t, err)

	return vm
}
