// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package profile

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestMLDSA65_DefaultSignVerify(t *testing.T) {
	s, err := NewMLDSA65Signer(rand.Reader)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	if s.Scheme() != SchemeMLDSA65 {
		t.Fatalf("scheme = %s, want ml-dsa-65", s.Scheme())
	}

	msg := []byte("price feed observation")
	sig, err := s.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if len(sig) != MLDSA65SignatureSize {
		t.Fatalf("sig size %d != %d", len(sig), MLDSA65SignatureSize)
	}

	if err := Verify(Default(), SchemeMLDSA65, s.PublicKey(), msg, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestStrictPQ_RefusesEd25519(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	err = Verify(Default(), SchemeEd25519, pub, []byte("x"), make([]byte, ed25519.SignatureSize))
	if err != ErrClassicalRefused {
		t.Fatalf("strict-PQ: expected ErrClassicalRefused, got %v", err)
	}
}

func TestLegacyEnabled_PermitsEd25519(t *testing.T) {
	pub, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	s := NewEd25519Signer(sk)
	msg := []byte("classical observation")
	sig, err := s.Sign(msg)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	legacy := Policy{LegacyClassicalEnabled: true}
	if err := Verify(legacy, SchemeEd25519, pub, msg, sig); err != nil {
		t.Fatalf("verify legacy: %v", err)
	}
}

func TestMLDSA65_TamperedSigRejected(t *testing.T) {
	s, _ := NewMLDSA65Signer(rand.Reader)
	msg := []byte("tamper target")
	sig, _ := s.Sign(msg)
	sig[0] ^= 0x01
	if err := Verify(Default(), SchemeMLDSA65, s.PublicKey(), msg, sig); err == nil {
		t.Fatalf("expected tampered ml-dsa-65 sig to fail")
	}
}

func TestPermit_UnknownScheme(t *testing.T) {
	if err := Default().Permit(Scheme(0xFF)); err == nil {
		t.Fatalf("expected unknown scheme to fail")
	}
}
