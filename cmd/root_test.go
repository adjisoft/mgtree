package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mgtree/internal/config"
)

func TestNormalizeArgsSupportsICAlias(t *testing.T) {
	got := normalizeArgs([]string{"-IC", "-s", "config"})
	want := []string{"--icons", "-s", "config"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("normalized args mismatch: got %v want %v", got, want)
	}
}

func TestDetectSortModeFromCombinedFlags(t *testing.T) {
	mode := detectSortMode([]string{"-alhXSr"})
	if mode != config.SortSize {
		t.Fatalf("expected last sort flag to win, got %s", mode)
	}
}

func TestRootCommandRejectsFastWithPreview(t *testing.T) {
	args := []string{"--fast", "--preview", "2"}
	command := NewRootCommand(args)
	command.SetArgs(args)
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "--fast cannot be combined with --preview") {
		t.Fatalf("expected fast/preview validation error, got %v", err)
	}
}

func TestRootCommandSupportsLsCluster(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{"-lah", "--no-color", root}
	command := NewRootCommand(args)
	output := &bytes.Buffer{}
	command.SetArgs(args)
	command.SetOut(output)
	command.SetErr(&bytes.Buffer{})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !strings.Contains(output.String(), "main.go") {
		t.Fatalf("expected rendered file in output, got %q", output.String())
	}
	if !strings.Contains(output.String(), "-rw") && !strings.Contains(output.String(), "-rwx") {
		t.Fatalf("expected long listing metadata in output, got %q", output.String())
	}
}

func TestRootCommandHelpUsesLongFlag(t *testing.T) {
	args := []string{"--help"}
	command := NewRootCommand(args)
	output := &bytes.Buffer{}
	command.SetArgs(args)
	command.SetOut(output)
	command.SetErr(&bytes.Buffer{})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	if !strings.Contains(output.String(), "human-readable") {
		t.Fatalf("expected help output, got %q", output.String())
	}
}
