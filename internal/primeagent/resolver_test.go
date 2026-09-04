package primeagent

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const listFixture = `{"sessions":[
 {"sessionName":"aop-3398-conductor","sessionId":"01a05d8a-238b-7469-9078-706014ce4015","sessionFile":"/w/s/01a05d8a.jsonl","cwd":"/workspace","runtimeKind":"top-level"},
 {"sessionName":"aop-3398","sessionId":"01a06311-0b5a-7492-b941-8f028e193050","sessionFile":"/w/s/01a06311.jsonl","cwd":"/workspace/.worktrees/workspace/aop-3398","runtimeKind":"top-level"},
 {"sessionName":"aop-4883-conductor","sessionId":"01a06399-98d3-76ba-a041-f1a5ccd1f257","sessionFile":"/w/s/01a06399.jsonl","cwd":"/workspace","runtimeKind":"top-level"},
 {"sessionName":"spike-repo","sessionId":"01a0ffff-0000-7000-8000-000000000000","sessionFile":"/w/s/sub.jsonl","cwd":"/workspace","runtimeKind":"subagent"}
]}`

func TestParseListKeepsOnlyTopLevelSessions(t *testing.T) {
	sessions, err := parseList([]byte(listFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 3 {
		t.Fatalf("sessions = %#v, want 3 top-level", sessions)
	}
	for _, session := range sessions {
		if session.Name == "spike-repo" {
			t.Fatal("subagent session was kept")
		}
	}
}

func TestNameFromTitle(t *testing.T) {
	cases := map[string]string{
		"prime-agent - aop-3398 - aop-3398":            "aop-3398",
		"prime-agent - aop-3398-conductor - workspace": "aop-3398-conductor",
		"prime-agent - Agents":                         "",
		"claude - something - dir":                     "",
		"":                                             "",
	}
	for title, want := range cases {
		if got := NameFromTitle(title); got != want {
			t.Errorf("NameFromTitle(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestIsPrime(t *testing.T) {
	for _, agent := range []string{"prime-agent", "Prime-Agent", " primeagent ", "prime"} {
		if !IsPrime(agent) {
			t.Errorf("IsPrime(%q) = false", agent)
		}
	}
	for _, agent := range []string{"pi", "claude", "", "prime-time"} {
		if IsPrime(agent) {
			t.Errorf("IsPrime(%q) = true", agent)
		}
	}
}

func TestLookupPrefersNameThenUniqueCwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake binary is a shell script")
	}
	dir := t.TempDir()
	fixture := filepath.Join(dir, "list.json")
	if err := os.WriteFile(fixture, []byte(listFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "prime-agent")
	script := "#!/bin/sh\ncat \"" + fixture + "\"\n"
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	previous := queryTimeout
	queryTimeout = 30 * time.Second
	t.Cleanup(func() { queryTimeout = previous })
	resolver := &Resolver{binary: binary}
	ctx := context.Background()

	byName, ok := resolver.Lookup(ctx, "aop-3398", "")
	if !ok || byName.ID != "01a06311-0b5a-7492-b941-8f028e193050" {
		t.Fatalf("lookup by name = %#v, %v", byName, ok)
	}
	byCwd, ok := resolver.Lookup(ctx, "", "/workspace/.worktrees/workspace/aop-3398")
	if !ok || byCwd.Name != "aop-3398" {
		t.Fatalf("lookup by unique cwd = %#v, %v", byCwd, ok)
	}
	if _, ok := resolver.Lookup(ctx, "", "/workspace"); ok {
		t.Fatal("ambiguous cwd (two conductors) must not resolve")
	}
	if _, ok := resolver.Lookup(ctx, "nobody", "/nowhere"); ok {
		t.Fatal("unknown name and cwd resolved")
	}
}

func TestLookupWithoutBinaryFailsClosed(t *testing.T) {
	resolver := &Resolver{binary: filepath.Join(t.TempDir(), "missing-prime-agent")}
	if _, ok := resolver.Lookup(context.Background(), "aop-3398", "/workspace"); ok {
		t.Fatal("resolved without a prime-agent binary")
	}
}
