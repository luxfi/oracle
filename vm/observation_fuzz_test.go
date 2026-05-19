// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/json"
	"testing"
)

// FuzzObservationDecode feeds the JSON observation decoder arbitrary bytes
// to confirm it never panics — the decoder is at the trust boundary between
// O-Chain RPC callers and the operator key registry.
func FuzzObservationDecode(f *testing.F) {
	good := &Observation{
		Value:     []byte("100.0"),
		Scheme:    0x01,
		Signature: []byte("seed-sig"),
	}
	if b, err := json.Marshal(good); err == nil {
		f.Add(b)
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"scheme":255}`))
	f.Add([]byte(`{"signature":null}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var o Observation
		_ = json.Unmarshal(data, &o) // must not panic
	})
}

// FuzzOracleRecordDecode does the same for executor records.
func FuzzOracleRecordDecode(f *testing.F) {
	good := &OracleRecord{
		Endpoint:   "https://x.test/v1",
		ResultCode: 200,
		Scheme:     0x01,
	}
	if b, err := json.Marshal(good); err == nil {
		f.Add(b)
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"resultCode":-1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var r OracleRecord
		_ = json.Unmarshal(data, &r) // must not panic
	})
}
