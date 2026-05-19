// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package zaptransport wraps github.com/luxfi/zap as the intra-Lux operator
// transport for oracled.
//
// Decomplecting: oracled talks to TWO surfaces.
//
//  1. External sources (Bitcoin RPC, Ethereum RPC, Chainlink, Pyth, market
//     data APIs) — HTTP / JSON / native chain primitive. This is NOT a
//     ZAP surface.
//
//  2. Other oracle operators within Lux — ZAP. ZAP carries sealed,
//     PQ-TLS-1.3-friendly frames between operator nodes for cross-operator
//     Observation gossip and OracleRecord broadcast before O-Chain has
//     committed them.
//
// This package is purely a transport — verification is the VM's job and
// stays gated by pkg/profile.
package zaptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/luxfi/zap"
)

// Message types reserved on the oracle ZAP plane. Wire-stable enum.
//
// ZAP encodes the message type in the upper byte of the wire flags field
// (`msgType << 8`), so the type space is 1 byte. We carve out 0x58..0x5F
// for oracle-plane traffic. Values are uint16 to match zap.Node.Handle.
const (
	// MsgOracleHello — opening handshake.
	MsgOracleHello uint16 = 0x58

	// MsgOracleObservation — operator-to-operator pre-commit broadcast of
	// a signed Observation. The payload is the JSON-encoded Observation.
	// The receiving operator verifies it through VM.VerifyObservationSignature
	// before doing anything else with it.
	MsgOracleObservation uint16 = 0x59

	// MsgOracleRecord — executor-to-executor broadcast of a signed
	// OracleRecord (external write/read result).
	MsgOracleRecord uint16 = 0x5A
)

// ServiceType is the mDNS service tag for oracle operators.
const ServiceType = "_luxd-oracle._tcp"

// Config configures an oracle-plane ZAP node.
type Config struct {
	NodeID      string
	Port        int
	NoDiscovery bool
	Logger      *slog.Logger
}

// Node is a thin facade over zap.Node so the oracle package never imports
// the ZAP wire layer directly.
type Node struct {
	z   *zap.Node
	log *slog.Logger
}

// New constructs an oracle-plane ZAP node. Start must be called before use.
func New(cfg Config) (*Node, error) {
	if cfg.NodeID == "" {
		return nil, errors.New("zaptransport: NodeID required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	z := zap.NewNode(zap.NodeConfig{
		NodeID:      cfg.NodeID,
		ServiceType: ServiceType,
		Port:        cfg.Port,
		NoDiscovery: cfg.NoDiscovery,
		Logger:      cfg.Logger,
	})
	return &Node{z: z, log: cfg.Logger}, nil
}

// Start brings the node online.
func (n *Node) Start() error { return n.z.Start() }

// Stop shuts the node down.
func (n *Node) Stop() { n.z.Stop() }

// NodeID returns this operator's ZAP node identifier.
func (n *Node) NodeID() string { return n.z.NodeID() }

// Peers lists currently connected oracle operator peers.
func (n *Node) Peers() []string { return n.z.Peers() }

// HandleObservation installs h as the consumer of inbound MsgOracleObservation
// frames. The handler receives the JSON-encoded Observation bytes; it is
// the handler's job to decode and run them through VM.VerifyObservationSignature.
func (n *Node) HandleObservation(h func(ctx context.Context, from string, observation []byte) error) {
	n.z.Handle(MsgOracleObservation, func(ctx context.Context, from string, msg *zap.Message) (*zap.Message, error) {
		body, err := jsonPayload(msg)
		if err != nil {
			return nil, err
		}
		if err := h(ctx, from, body); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

// HandleRecord installs h as the consumer of inbound MsgOracleRecord frames.
func (n *Node) HandleRecord(h func(ctx context.Context, from string, record []byte) error) {
	n.z.Handle(MsgOracleRecord, func(ctx context.Context, from string, msg *zap.Message) (*zap.Message, error) {
		body, err := jsonPayload(msg)
		if err != nil {
			return nil, err
		}
		if err := h(ctx, from, body); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

// BroadcastObservation sends observation bytes to every connected peer.
// Verification happens at the receiving end via profile.Verify; this
// transport does not run any signature math itself.
func (n *Node) BroadcastObservation(ctx context.Context, observation []byte) map[string]error {
	msg, err := build(MsgOracleObservation, observation)
	if err != nil {
		return map[string]error{"_build": err}
	}
	return n.z.Broadcast(ctx, msg)
}

// BroadcastRecord sends record bytes to every connected peer.
func (n *Node) BroadcastRecord(ctx context.Context, record []byte) map[string]error {
	msg, err := build(MsgOracleRecord, record)
	if err != nil {
		return map[string]error{"_build": err}
	}
	return n.z.Broadcast(ctx, msg)
}

// ConnectDirect adds an unannounced peer (e.g. via static config).
func (n *Node) ConnectDirect(addr string) error { return n.z.ConnectDirect(addr) }

// jsonPayload extracts the payload from a ZAP message.
func jsonPayload(msg *zap.Message) ([]byte, error) {
	root := msg.Root()
	if root.IsNull() {
		return nil, errors.New("zaptransport: empty zap message")
	}
	b := root.Bytes(0)
	if len(b) == 0 {
		return nil, errors.New("zaptransport: zero-length payload")
	}
	var probe map[string]any
	if err := json.Unmarshal(b, &probe); err != nil {
		return nil, fmt.Errorf("zaptransport: payload not JSON: %w", err)
	}
	return b, nil
}

// build frames body bytes as a ZAP message of the given type.
func build(msgType uint16, body []byte) (*zap.Message, error) {
	b := zap.NewBuilder(len(body) + 64)
	ob := b.StartObject(8)
	ob.SetBytes(0, body)
	ob.FinishAsRoot()
	flags := msgType << 8
	data := b.FinishWithFlags(flags)
	return zap.Parse(data)
}
