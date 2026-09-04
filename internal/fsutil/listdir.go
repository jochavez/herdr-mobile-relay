package fsutil

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type DirListing struct {
	Current struct {
		Path  string `json:"path"`
		Label string `json:"label"`
	} `json:"current"`
	Parent      string     `json:"parent"`
	Directories []DirEntry `json:"directories"`
}

// ListDirectories lists the browsable subdirectories of path, confined to home.
func ListDirectories(path, home string) DirListing {
	return ListDirectoriesWithin(path, home, nil)
}

// ListDirectoriesWithin confines browsing to home plus extraRoots, each an
// absolute directory outside home that the operator chose to expose. A path
// outside every root falls back to the home listing, which also offers each
// extra root as an entry; an extra root's parent is home, so the browser can
// always climb back. Directories is never nil: the phone requires an array.
func ListDirectoriesWithin(path, home string, extraRoots []string) DirListing {
	home = canonicalDirectory(home)
	roots := canonicalExtraRoots(home, extraRoots)
	if path == "" {
		path = home
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(home, path)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
		path = resolved
	}
	root := containingRoot(path, home, roots)
	if root == "" {
		root, path = home, home
	}
	listing, ok := listWithinRoot(path, root, home, roots)
	if !ok && path != home {
		listing, _ = listWithinRoot(home, home, home, roots)
	}
	return listing
}

func listWithinRoot(path, root, home string, roots []string) (DirListing, bool) {
	result := DirListing{Directories: []DirEntry{}}
	result.Current.Path = path
	result.Current.Label = displayPath(path, home)
	switch {
	case path == home:
	case path == root:
		result.Parent = home
	default:
		result.Parent = filepath.Dir(path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return result, false
	}
	opened, err := os.OpenRoot(root)
	if err != nil {
		return result, false
	}
	defer opened.Close()
	directory, err := opened.Open(relative)
	if err != nil {
		return result, false
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil || !info.IsDir() {
		return result, false
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return result, true
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		childPath, err := filepath.EvalSymlinks(filepath.Join(path, name))
		if err != nil || !within(root, childPath) {
			continue
		}
		childInfo, statErr := os.Stat(childPath)
		if statErr != nil || !childInfo.IsDir() {
			continue
		}
		result.Directories = append(result.Directories, DirEntry{
			Name: name,
			Path: filepath.Join(path, name),
		})
	}
	if path == home {
		for _, extra := range roots {
			result.Directories = append(result.Directories, DirEntry{Name: extra, Path: extra})
		}
	}
	sort.Slice(result.Directories, func(i, j int) bool {
		return strings.ToLower(result.Directories[i].Name) < strings.ToLower(result.Directories[j].Name)
	})
	return result, true
}

func canonicalDirectory(path string) string {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}

func canonicalExtraRoots(home string, extraRoots []string) []string {
	var roots []string
	for _, raw := range extraRoots {
		if raw == "" || !filepath.IsAbs(raw) {
			continue
		}
		root := canonicalDirectory(raw)
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() || root == home || within(home, root) {
			continue
		}
		roots = append(roots, root)
	}
	return roots
}

func containingRoot(path, home string, roots []string) string {
	if path == home || within(home, path) {
		return home
	}
	for _, root := range roots {
		if path == root || within(root, path) {
			return root
		}
	}
	return ""
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func displayPath(path, home string) string {
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~/" + path[len(home)+1:]
	}
	return path
}
