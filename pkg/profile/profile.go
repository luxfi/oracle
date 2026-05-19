// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package profile carries the oracle's signing-profile decision in ONE place.
//
// Decomplecting principle: the oracle has TWO distinct signing surfaces.
//
//  1. Intra-Lux operator surface — signed Observations, OracleRecords,
//     OracleCommits, executor attestations to O-Chain RPC. Default = ML-DSA-65
//     (FIPS 204, NIST Level 3). Classical Ed25519 is opt-in only via
//     LegacyClassicalEnabled.
//
//  2. External-source surface — Bitcoin RPC, Ethereum RPC, Pyth, Chainlink,
//     market data APIs. These are NOT subject to the PQ default; they ride
//     whatever transport and authentication the external source provides.
//     The oracle never PQ-flips a fetch against a public price feed.
//
// All policy lives here; primitives must never re-decide the profile.
package profile

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/luxfi/crypto/mldsa"
)

// Scheme identifies the operator signing scheme. Wire-stable enum.
type Scheme uint8

const (
	// SchemeMLDSA65 — ML-DSA-65 (FIPS 204). Default for intra-Lux operator
	// signatures (signed Observations, OracleRecords, OracleCommits).
	SchemeMLDSA65 Scheme = 0x01
	// SchemeEd25519 — Classical Ed25519. Opt-in only via Policy.LegacyClassicalEnabled.
	SchemeEd25519 Scheme = 0x02
)

func (s Scheme) String() string {
	switch s {
	case SchemeMLDSA65:
		return "ml-dsa-65"
	case SchemeEd25519:
		return "ed25519"
	default:
		return fmt.Sprintf("scheme(0x%02x)", uint8(s))
	}
}

// Policy carries the oracle operator's signing-profile decision.
//
// Default value (zero-Policy) means: ML-DSA-65 only, classical refused.
// This is intentional — the safe default is strict-PQ.
type Policy struct {
	// LegacyClassicalEnabled, when true, allows Ed25519 keys and verifies
	// classical observations. Production deployments inside Lux should
	// leave this off.
	LegacyClassicalEnabled bool
}

// Default returns the strict-PQ policy: ML-DSA-65 only.
func Default() Policy { return Policy{} }

// Permit reports whether a scheme is currently accepted under p.
// This is the single function classical primitives consult before doing
// anything with classical key material.
func (p Policy) Permit(s Scheme) error {
	switch s {
	case SchemeMLDSA65:
		return nil
	case SchemeEd25519:
		if !p.LegacyClassicalEnabled {
			return ErrClassicalRefused
		}
		return nil
	default:
		return fmt.Errorf("profile: unknown scheme %s", s)
	}
}

// ContextTag is the domain-separation tag bound into every operator
// signature so an Oracle observation cannot be replayed as some other
// Lux artifact and vice versa.
const ContextTag = "luxfi.oracle.v1"

// Default key/sig sizes for the active default scheme (ML-DSA-65).
const (
	MLDSA65PublicKeySize = mldsa.MLDSA65PublicKeySize
	MLDSA65SignatureSize = mldsa.MLDSA65SignatureSize
)

// ErrClassicalRefused is returned when an Ed25519 signature is presented
// under a strict-PQ policy.
var ErrClassicalRefused = errors.New("profile: classical scheme refused under strict-PQ")

// Signer carries an operator's signing key. Construction picks the scheme;
// Sign always domain-separates with ContextTag.
type Signer struct {
	scheme Scheme
	mldsa  *mldsa.PrivateKey
	ed     ed25519.PrivateKey
}

// NewMLDSA65Signer returns a fresh ML-DSA-65 signer.
func NewMLDSA65Signer(rng io.Reader) (*Signer, error) {
	if rng == nil {
		rng = rand.Reader
	}
	sk, err := mldsa.GenerateKey(rng, mldsa.MLDSA65)
	if err != nil {
		return nil, err
	}
	return &Signer{scheme: SchemeMLDSA65, mldsa: sk}, nil
}

// NewMLDSA65SignerFromBytes restores an ML-DSA-65 signer from its serialised
// secret-key bytes.
func NewMLDSA65SignerFromBytes(skBytes []byte) (*Signer, error) {
	sk, err := mldsa.PrivateKeyFromBytes(mldsa.MLDSA65, skBytes)
	if err != nil {
		return nil, err
	}
	return &Signer{scheme: SchemeMLDSA65, mldsa: sk}, nil
}

// NewEd25519Signer wraps a classical Ed25519 key. Callers must have already
// consulted Policy.Permit(SchemeEd25519); this constructor does not.
func NewEd25519Signer(sk ed25519.PrivateKey) *Signer {
	return &Signer{scheme: SchemeEd25519, ed: sk}
}

// Scheme returns the underlying scheme tag.
func (s *Signer) Scheme() Scheme { return s.scheme }

// PublicKey returns the serialised public key bytes.
func (s *Signer) PublicKey() []byte {
	switch s.scheme {
	case SchemeMLDSA65:
		return s.mldsa.PublicKey.Bytes()
	case SchemeEd25519:
		return s.ed.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

// Sign produces a signature over msg with the oracle's domain-separation
// context (FIPS 204 §5.2 ctx for ML-DSA, prepended-tag for Ed25519).
func (s *Signer) Sign(msg []byte) ([]byte, error) {
	switch s.scheme {
	case SchemeMLDSA65:
		return s.mldsa.SignCtx(rand.Reader, msg, []byte(ContextTag))
	case SchemeEd25519:
		// Ed25519 has no native ctx; we prepend the tag.
		tagged := make([]byte, 0, len(ContextTag)+len(msg))
		tagged = append(tagged, ContextTag...)
		tagged = append(tagged, msg...)
		return ed25519.Sign(s.ed, tagged), nil
	default:
		return nil, fmt.Errorf("profile: signer scheme %s not implemented", s.scheme)
	}
}

// Verify checks sig over msg under pub for the given scheme, gated by p.
// This is the SINGLE place all oracle observation/record verification
// flows through.
func Verify(p Policy, scheme Scheme, pub, msg, sig []byte) error {
	if err := p.Permit(scheme); err != nil {
		return err
	}
	switch scheme {
	case SchemeMLDSA65:
		if len(pub) != MLDSA65PublicKeySize {
			return fmt.Errorf("profile: ml-dsa-65 pubkey size %d != %d", len(pub), MLDSA65PublicKeySize)
		}
		pk, err := mldsa.PublicKeyFromBytes(pub, mldsa.MLDSA65)
		if err != nil {
			return err
		}
		if !pk.VerifySignatureCtx(msg, sig, []byte(ContextTag)) {
			return errors.New("profile: ml-dsa-65 signature invalid")
		}
		return nil
	case SchemeEd25519:
		if len(pub) != ed25519.PublicKeySize {
			return fmt.Errorf("profile: ed25519 pubkey size %d != %d", len(pub), ed25519.PublicKeySize)
		}
		tagged := make([]byte, 0, len(ContextTag)+len(msg))
		tagged = append(tagged, ContextTag...)
		tagged = append(tagged, msg...)
		if !ed25519.Verify(ed25519.PublicKey(pub), tagged, sig) {
			return errors.New("profile: ed25519 signature invalid")
		}
		return nil
	default:
		return fmt.Errorf("profile: unknown scheme %s", scheme)
	}
}
