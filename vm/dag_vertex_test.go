// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"testing"

	"github.com/luxfi/consensus/core/choices"
	"github.com/luxfi/ids"
)

func TestOracleVertexConflicts_SameFeedRound(t *testing.T) {
	feedID := ids.GenerateTestID()

	v1 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: feedID, Round: 42}},
	}
	v2 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: feedID, Round: 42}},
	}

	if !v1.Conflicts(v2) {
		t.Fatal("expected conflict: same feedID + round")
	}
	if !v2.Conflicts(v1) {
		t.Fatal("expected conflict: symmetric check failed")
	}
}

func TestOracleVertexConflicts_DifferentFeeds(t *testing.T) {
	v1 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: ids.GenerateTestID(), Round: 42}},
	}
	v2 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: ids.GenerateTestID(), Round: 42}},
	}

	if v1.Conflicts(v2) {
		t.Fatal("expected no conflict: different feeds commute even at same round")
	}
}

func TestOracleVertexConflicts_SameFeedDifferentRound(t *testing.T) {
	feedID := ids.GenerateTestID()

	v1 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: feedID, Round: 1}},
	}
	v2 := &OracleVertex{
		id:     ids.GenerateTestID(),
		status: choices.Processing,
		keys:   []FeedRoundKey{{FeedID: feedID, Round: 2}},
	}

	if v1.Conflicts(v2) {
		t.Fatal("expected no conflict: same feed but different rounds")
	}
}
