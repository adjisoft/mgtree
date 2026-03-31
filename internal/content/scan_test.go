package content

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestInspectFileMatchesContentAndPreview(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.txt")
	data := "alpha\nsecret-token\nomega\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := InspectFile(Request{
		Path:             path,
		PreviewLines:     2,
		ContentExpr:      regexp.MustCompile("secret"),
		MaxScanBytes:     1 << 20,
		PreviewReadBytes: 64 << 10,
	})
	if err != nil {
		t.Fatalf("inspect file: %v", err)
	}
	if !result.Matched {
		t.Fatal("expected content match")
	}
	if len(result.Preview) != 2 {
		t.Fatalf("expected 2 preview lines, got %d", len(result.Preview))
	}
}

func TestInspectFileSkipsLargeByDefault(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.txt")
	if err := os.WriteFile(path, bytes.Repeat([]byte("a"), 32), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := InspectFile(Request{
		Path:         path,
		MaxScanBytes: 8,
	})
	if err != nil {
		t.Fatalf("inspect file: %v", err)
	}
	if !result.SkippedLarge {
		t.Fatal("expected large file skip")
	}
}

func TestInspectFileSmartPreviewSkipsLeadingComments(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	data := "// comment\n\npackage main\nimport \"fmt\"\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := InspectFile(Request{
		Path:             path,
		PreviewLines:     2,
		MaxScanBytes:     1 << 20,
		PreviewReadBytes: 64 << 10,
	})
	if err != nil {
		t.Fatalf("inspect file: %v", err)
	}
	if len(result.Preview) == 0 || result.Preview[0].Text != "package main" {
		t.Fatalf("expected smart preview to start at package line, got %+v", result.Preview)
	}
}
