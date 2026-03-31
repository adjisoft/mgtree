package walk

import (
	"os"
	"path/filepath"
	"testing"

	"mgtree/internal/config"
)

func TestBuildHidesDotfilesByDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	prepared, err := config.Prepare(config.Options{Root: root, Depth: -1}, map[string]string{}, false, "linux", config.SortDefault)
	if err != nil {
		t.Fatalf("prepare options: %v", err)
	}

	tree, _, err := Build(prepared)
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Name != "main.go" {
		t.Fatalf("expected hidden file to be skipped, got %+v", tree.Children)
	}
}

func TestBuildShowsDotfilesWithAll(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	prepared, err := config.Prepare(config.Options{Root: root, Depth: -1, All: true}, map[string]string{}, false, "linux", config.SortDefault)
	if err != nil {
		t.Fatalf("prepare options: %v", err)
	}

	tree, _, err := Build(prepared)
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	if len(tree.Children) != 2 {
		t.Fatalf("expected hidden file to be shown, got %+v", tree.Children)
	}
}

func TestBuildDirectoryOnlyShowsRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	prepared, err := config.Prepare(config.Options{Root: root, Depth: -1, DirectoryOnly: true}, map[string]string{}, false, "linux", config.SortDefault)
	if err != nil {
		t.Fatalf("prepare options: %v", err)
	}

	tree, _, err := Build(prepared)
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	if len(tree.Children) != 0 {
		t.Fatalf("expected directory-only mode to suppress children, got %+v", tree.Children)
	}
}

func TestBuildSortsBySize(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "small.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), []byte("abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}

	prepared, err := config.Prepare(config.Options{Root: root, Depth: -1}, map[string]string{}, false, "linux", config.SortSize)
	if err != nil {
		t.Fatalf("prepare options: %v", err)
	}

	tree, _, err := Build(prepared)
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	if len(tree.Children) != 2 || tree.Children[0].Name != "large.txt" {
		t.Fatalf("expected size sort descending, got %+v", tree.Children)
	}
}
