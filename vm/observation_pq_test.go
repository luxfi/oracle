// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/luxfi/ids"
	"github.com/luxfi/oracle/pkg/profile"
)

func newObs(t *testing.T, feedID ids.ID, opID ids.NodeID) *Observation {
	t.Helper()
	var meta [32]byte
	if _, err := rand.Read(meta[:]); err != nil {
		t.Fatalf("rand meta: %v", err)
	}
	return &Observation{
		FeedID:     feedID,
		Value:      []byte("123.456"),
		Timestamp:  time.Unix(1700000000, 0),
		SourceMeta: meta,
		OperatorID: opID,
	}
}

// TestE2E_PQObservation_DefaultMLDSA65 exercises the default (strict-PQ)
// path: an operator with an ML-DSA-65 key registers, signs an Observation,
// and the VM accepts it.
func TestE2E_PQObservation_DefaultMLDSA65(t *testing.T) {
	vm := &VM{
		operatorKeys: make(map[ids.NodeID]operatorKey),
		policy:       profile.Default(),
	}

	signer, err := profile.NewMLDSA65Signer(rand.Reader)
	if err != nil {
		t.Fatalf("new ml-dsa-65 signer: %v", err)
	}

	var opID ids.NodeID
	if _, err := rand.Read(opID[:]); err != nil {
		t.Fatalf("rand op id: %v", err)
	}
	if err := vm.RegisterOperatorKey(opID, profile.SchemeMLDSA65, signer.PublicKey()); err != nil {
		t.Fatalf("register pq key: %v", err)
	}

	obs := newObs(t, ids.GenerateTestID(), opID)
	obs.Scheme = profile.SchemeMLDSA65
	sig, err := signer.Sign(observationMessage(obs))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	obs.Signature = sig

	if err := vm.VerifyObservationSignature(obs); err != nil {
		t.Fatalf("strict-PQ observation verify: %v", err)
	}
}

// TestStrictPQ_RefusesEd25519Observation asserts the policy gate fires
// BEFORE any classical signature math: an Ed25519 observation under
// default policy is refused.
func TestStrictPQ_RefusesEd25519Observation(t *testing.T) {
	vm := &VM{
		operatorKeys: make(map[ids.NodeID]operatorKey),
		policy:       profile.Default(),
	}
	pub, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	var opID ids.NodeID
	if _, err := rand.Read(opID[:]); err != nil {
		t.Fatalf("rand op id: %v", err)
	}
	if err := vm.RegisterOperatorKey(opID, profile.SchemeEd25519, pub); err != nil {
		t.Fatalf("register classical key: %v", err)
	}

	obs := newObs(t, ids.GenerateTestID(), opID)
	obs.Scheme = profile.SchemeEd25519
	tagged := append([]byte(profile.ContextTag), observationMessage(obs)...)
	obs.Signature = ed25519.Sign(sk, tagged)

	if err := vm.VerifyObservationSignature(obs); err == nil {
		t.Fatalf("strict-PQ MUST refuse classical observation")
	}
}

// TestLegacyEnabled_AcceptsEd25519Observation confirms the opt-in toggle works.
func TestLegacyEnabled_AcceptsEd25519Observation(t *testing.T) {
	vm := &VM{
		operatorKeys: make(map[ids.NodeID]operatorKey),
		policy:       profile.Policy{LegacyClassicalEnabled: true},
	}
	pub, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	var opID ids.NodeID
	if _, err := rand.Read(opID[:]); err != nil {
		t.Fatalf("rand op id: %v", err)
	}
	if err := vm.RegisterOperatorKey(opID, profile.SchemeEd25519, pub); err != nil {
		t.Fatalf("register classical key: %v", err)
	}

	obs := newObs(t, ids.GenerateTestID(), opID)
	obs.Scheme = profile.SchemeEd25519
	tagged := append([]byte(profile.ContextTag), observationMessage(obs)...)
	obs.Signature = ed25519.Sign(sk, tagged)

	if err := vm.VerifyObservationSignature(obs); err != nil {
		t.Fatalf("legacy-enabled classical observation verify: %v", err)
	}
}

// TestE2E_PQRecord_DefaultMLDSA65 checks the executor-side record path
// under strict-PQ.
func TestE2E_PQRecord_DefaultMLDSA65(t *testing.T) {
	vm := &VM{
		operatorKeys: make(map[ids.NodeID]operatorKey),
		policy:       profile.Default(),
	}
	signer, err := profile.NewMLDSA65Signer(rand.Reader)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	var exec ids.NodeID
	if _, err := rand.Read(exec[:]); err != nil {
		t.Fatalf("rand exec: %v", err)
	}
	if err := vm.RegisterOperatorKey(exec, profile.SchemeMLDSA65, signer.PublicKey()); err != nil {
		t.Fatalf("register: %v", err)
	}

	var reqID [32]byte
	if _, err := rand.Read(reqID[:]); err != nil {
		t.Fatalf("rand reqID: %v", err)
	}
	r := &OracleRecord{
		RequestID:  reqID,
		Executor:   exec,
		Timestamp:  1700000000,
		Endpoint:   "https://api.example.test/v1/quote",
		BodyHash:   [32]byte{0xAB},
		ResultCode: 200,
		Scheme:     profile.SchemeMLDSA65,
	}
	sig, err := signer.Sign(recordMessage(r))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	r.Signature = sig

	if err := vm.VerifyRecordSignature(r); err != nil {
		t.Fatalf("strict-PQ record verify: %v", err)
	}
}

// TestUnregisteredOperator_Refused asserts unregistered keys always fail.
func TestUnregisteredOperator_Refused(t *testing.T) {
	vm := &VM{
		operatorKeys: make(map[ids.NodeID]operatorKey),
		policy:       profile.Default(),
	}
	obs := newObs(t, ids.GenerateTestID(), ids.GenerateTestNodeID())
	obs.Scheme = profile.SchemeMLDSA65
	obs.Signature = []byte("anything")
	if err := vm.VerifyObservationSignature(obs); err == nil {
		t.Fatalf("unregistered operator MUST be refused")
	}
}
