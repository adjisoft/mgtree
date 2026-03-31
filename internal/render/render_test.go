package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"mgtree/internal/config"
	"mgtree/internal/model"
)

func TestRenderPlainAndColorModes(t *testing.T) {
	root := &model.Node{
		Name:    "root",
		IsDir:   true,
		Mode:    0o755 | 0o040000,
		ModTime: time.Unix(1700000000, 0),
		Children: []*model.Node{
			{Name: "main.go", RelativePath: "main.go", Mode: 0o644, ModTime: time.Unix(1700000000, 0)},
		},
	}

	plain := New(config.Prepared{UseColor: false, UseIcons: false})
	plainBuffer := &bytes.Buffer{}
	if err := plain.Render(plainBuffer, root, model.Stats{}); err != nil {
		t.Fatalf("render plain: %v", err)
	}
	if strings.Contains(plainBuffer.String(), "\x1b[") {
		t.Fatal("plain render should not contain ANSI escape codes")
	}

	coloredRoot := &model.Node{
		Name:    "root",
		IsDir:   true,
		Mode:    0o755 | 0o040000,
		ModTime: time.Unix(1700000000, 0),
		Children: []*model.Node{
			{Name: "main.go", RelativePath: "main.go", NameHighlights: []model.Range{{Start: 0, End: 4}}, Mode: 0o644, ModTime: time.Unix(1700000000, 0)},
		},
	}
	colored := New(config.Prepared{UseColor: true, UseIcons: false})
	colorBuffer := &bytes.Buffer{}
	if err := colored.Render(colorBuffer, coloredRoot, model.Stats{}); err != nil {
		t.Fatalf("render colored: %v", err)
	}
	if !strings.Contains(colorBuffer.String(), "\x1b[") {
		t.Fatal("colored render should contain ANSI escape codes")
	}
}

func TestRenderLongHumanReadableAndClassify(t *testing.T) {
	root := &model.Node{
		Name:       "script.sh",
		Mode:       0o755,
		Size:       2048,
		Executable: true,
		Classifier: "*",
		ModTime:    time.Unix(1700000000, 0),
	}
	output := &bytes.Buffer{}
	renderer := New(config.Prepared{
		Options:  config.Options{Long: true, HumanReadable: true, Classify: true},
		UseIcons: false,
		UseColor: false,
	})
	if err := renderer.Render(output, root, model.Stats{}); err != nil {
		t.Fatalf("render long classify: %v", err)
	}
	rendered := output.String()
	if !strings.Contains(rendered, "2.0KB") || !strings.Contains(rendered, "script.sh*") {
		t.Fatalf("unexpected render output %q", rendered)
	}
}
