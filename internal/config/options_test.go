package config

import "testing"

func TestPrepareRejectsInvalidRegex(t *testing.T) {
	_, err := Prepare(Options{RegexPattern: "["}, map[string]string{}, false, "linux", SortDefault)
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
}

func TestPrepareResolvesModesAndSort(t *testing.T) {
	prepared, err := Prepare(
		Options{},
		map[string]string{"LANG": "en_US.UTF-8", "TERM": "xterm-256color"},
		true,
		"linux",
		SortTime,
	)
	if err != nil {
		t.Fatalf("prepare options: %v", err)
	}
	if !prepared.UseColor {
		t.Fatal("expected color auto detection to enable colors")
	}
	if !prepared.UseIcons {
		t.Fatal("expected icon auto detection to enable icons")
	}
	if prepared.SortMode != SortTime {
		t.Fatalf("expected sort mode time, got %s", prepared.SortMode)
	}
}

func TestPrepareRejectsFastWithContent(t *testing.T) {
	_, err := Prepare(Options{Fast: true, ContentPattern: "token"}, map[string]string{}, false, "linux", SortDefault)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
