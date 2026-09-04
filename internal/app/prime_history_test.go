package app

import (
	"testing"

	"github.com/0cv/herdr-mobile-relay/internal/coordinator"
)

func TestStitchesTerminalHistoryForPrimeLikeClaudeAndQoder(t *testing.T) {
	for _, agent := range []string{"claude", "Claude Code", "qoder", "prime-agent", "Prime-Agent"} {
		if !stitchesTerminalHistory(agent) {
			t.Errorf("stitchesTerminalHistory(%q) = false", agent)
		}
	}
	for _, agent := range []string{"pi", "codex", "opencode", ""} {
		if stitchesTerminalHistory(agent) {
			t.Errorf("stitchesTerminalHistory(%q) = true", agent)
		}
	}
}

func TestPreparePaneResponseStitchesPrimeScrollback(t *testing.T) {
	s := testServerWithCacheDir(t.TempDir())
	s.state.CommitInventory([]*coordinator.AgentState{{
		PaneID: "pane-1", Agent: "prime-agent", Status: "idle",
	}}, s.state.RevisionCounter())
	baseline := "history 1\nhistory 2\nhistory 3\nhistory 4\nhistory 5\nhistory 6\nhistory 7\nhistory 8"
	s.historyM.Merge("pane-1", baseline)
	response := map[string]any{
		"type": "pane_content", "pane_id": "pane-1",
		"content": "current 1\ncurrent 2", "format": "ansi",
		"truncated": false, "viewport_only": false,
	}
	s.preparePaneResponse(map[string]any{"pane_id": "pane-1", "lines": 100}, response)
	want := "history 1\nhistory 2\ncurrent 1\ncurrent 2"
	if response["content"] != want {
		t.Fatalf("prime pane content = %q, want stitched history %q", response["content"], want)
	}
}
