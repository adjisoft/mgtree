package filter

import (
	"regexp"
	"testing"
	"time"

	"mgtree/internal/config"
	"mgtree/internal/model"
)

func TestEvaluateMixedMode(t *testing.T) {
	prepared := config.Prepared{
		IncludeRegex: regexp.MustCompile(`.*\.go$`),
		SearchFolded: "auth",
		ContentRegex: regexp.MustCompile("token"),
	}

	eval := Evaluate(prepared, "auth_handler.go", "src/auth_handler.go", false)
	if !eval.SelfMatched {
		t.Fatal("expected self matched from regex and search")
	}

	eval = Evaluate(prepared, "main.go", "src/main.go", true)
	if !eval.SelfMatched {
		t.Fatal("expected content match to satisfy search/content stage")
	}
}

func TestMatchesExclude(t *testing.T) {
	expr := regexp.MustCompile(`node_modules|\.git`)
	if !MatchesExclude(expr, "src/.git/config", "config") {
		t.Fatal("expected exclude match")
	}
}

func TestSortChildrenModes(t *testing.T) {
	now := time.Now()
	nodes := []*model.Node{
		{Name: "b.txt", Size: 10, ModTime: now.Add(-time.Hour)},
		{Name: "a.go", Size: 100, ModTime: now},
	}
	SortChildren(nodes, config.SortSize, false)
	if nodes[0].Name != "a.go" {
		t.Fatalf("expected size sort descending, got %s", nodes[0].Name)
	}
	SortChildren(nodes, config.SortDefault, true)
	if nodes[0].Name != "b.txt" {
		t.Fatalf("expected reverse default sort, got %s", nodes[0].Name)
	}
}
