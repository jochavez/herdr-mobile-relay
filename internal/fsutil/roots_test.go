package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// canonicalTempDir resolves the symlinked temp root some platforms hand out
// (macOS /var -> /private/var), matching what the listing reports.
func canonicalTempDir(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestListDirectoriesWithinSurfacesExtraRootsAtHome(t *testing.T) {
	home := canonicalTempDir(t)
	workspace := canonicalTempDir(t)
	mustMkdir(t, filepath.Join(home, "docs"))
	mustMkdir(t, filepath.Join(workspace, "proj"))
	result := ListDirectoriesWithin("", home, []string{workspace})
	if result.Current.Path != home || result.Parent != "" {
		t.Fatalf("home listing = %+v", result.Current)
	}
	var names, paths []string
	for _, entry := range result.Directories {
		names = append(names, entry.Name)
		paths = append(paths, entry.Path)
	}
	if len(result.Directories) != 2 || paths[0] != workspace && paths[1] != workspace {
		t.Fatalf("home listing entries = %v / %v, want docs plus the extra root %s", names, paths, workspace)
	}
}

func TestListDirectoriesWithinBrowsesAnExtraRoot(t *testing.T) {
	home := canonicalTempDir(t)
	workspace := canonicalTempDir(t)
	mustMkdir(t, filepath.Join(workspace, "proj", "sub"))
	root := ListDirectoriesWithin(workspace, home, []string{workspace})
	if root.Current.Path != workspace || root.Parent != home {
		t.Fatalf("extra root listing = %+v parent=%q, want the root with parent %s", root.Current, root.Parent, home)
	}
	if len(root.Directories) != 1 || root.Directories[0].Name != "proj" {
		t.Fatalf("extra root entries = %+v, want proj", root.Directories)
	}
	child := ListDirectoriesWithin(filepath.Join(workspace, "proj"), home, []string{workspace})
	if child.Parent != workspace || len(child.Directories) != 1 || child.Directories[0].Name != "sub" {
		t.Fatalf("child listing = %+v parent=%q", child.Directories, child.Parent)
	}
}

func TestListDirectoriesWithinNeverReturnsNilDirectories(t *testing.T) {
	home := canonicalTempDir(t)
	empty := canonicalTempDir(t)
	for _, path := range []string{empty, filepath.Join(home, "missing")} {
		result := ListDirectoriesWithin(path, home, []string{empty})
		if result.Directories == nil {
			t.Fatalf("listing of %s has nil directories; the phone requires an array", path)
		}
	}
}

func TestListDirectoriesWithinFallsBackToHomeOutsideEveryRoot(t *testing.T) {
	home := canonicalTempDir(t)
	workspace := canonicalTempDir(t)
	elsewhere := canonicalTempDir(t)
	result := ListDirectoriesWithin(elsewhere, home, []string{workspace})
	if result.Current.Path != home {
		t.Fatalf("outside path listed %q, want fallback to home %s", result.Current.Path, home)
	}
}
