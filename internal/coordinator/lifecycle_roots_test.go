package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCwdAcceptsConfiguredExtraRootAndItsChildren(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	child := filepath.Join(workspace, "proj")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	lifecycle := &Lifecycle{home: home, extraRoots: []string{workspace}}
	canonical, _ := filepath.EvalSymlinks(workspace)
	if resolved, err := lifecycle.ResolveCwd(workspace); err != nil || resolved != canonical {
		t.Fatalf("extra root itself: resolved=%q err=%v, want %q", resolved, err, canonical)
	}
	if resolved, err := lifecycle.ResolveCwd(child); err != nil || resolved != filepath.Join(canonical, "proj") {
		t.Fatalf("child of extra root: resolved=%q err=%v", resolved, err)
	}
}

func TestResolveCwdStillRejectsOutsideEveryRoot(t *testing.T) {
	lifecycle := &Lifecycle{home: t.TempDir(), extraRoots: []string{t.TempDir()}}
	if _, err := lifecycle.ResolveCwd(t.TempDir()); err == nil {
		t.Fatal("a directory outside home and every extra root was accepted")
	}
}

func TestResolveCwdStillRejectsHomeItselfWithExtraRoots(t *testing.T) {
	home := t.TempDir()
	lifecycle := &Lifecycle{home: home, extraRoots: []string{t.TempDir()}}
	if _, err := lifecycle.ResolveCwd(home); err == nil {
		t.Fatal("home itself was accepted as a launch cwd")
	}
}
