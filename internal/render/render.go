package render

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"

	"mgtree/internal/config"
	"mgtree/internal/model"
)

type Renderer struct {
	prepared       config.Prepared
	dirColor       *color.Color
	goColor        *color.Color
	jsonColor      *color.Color
	scriptColor    *color.Color
	highlightColor *color.Color
}

type columnWidths struct {
	size int
}

func New(prepared config.Prepared) Renderer {
	renderer := Renderer{
		prepared:       prepared,
		dirColor:       color.New(color.FgHiBlue),
		goColor:        color.New(color.FgCyan),
		jsonColor:      color.New(color.FgHiYellow),
		scriptColor:    color.New(color.FgGreen),
		highlightColor: color.New(color.FgHiRed, color.Bold),
	}
	color.NoColor = !prepared.UseColor
	return renderer
}

func (r Renderer) Render(w io.Writer, root *model.Node, stats model.Stats) error {
	if _, err := fmt.Fprintln(w, r.renderLine(root, columnWidths{size: runeWidth(r.sizeText(root.Size))})); err != nil {
		return err
	}
	if err := r.renderChildren(w, root.Children, ""); err != nil {
		return err
	}
	if r.prepared.ShowStats {
		_, err := fmt.Fprintf(
			w,
			"\nScanned Files: %d\nScanned Folders: %d\nMatched Files: %d\nMatched Folders: %d\nTotal Size: %s\n",
			stats.ScannedFiles,
			stats.ScannedFolders,
			stats.MatchedFiles,
			stats.MatchedFolders,
			r.sizeText(stats.TotalSize),
		)
		return err
	}
	return nil
}

func (r Renderer) renderChildren(w io.Writer, children []*model.Node, prefix string) error {
	widths := r.measureColumns(children)
	for idx, child := range children {
		if err := r.renderNode(w, child, prefix, idx == len(children)-1, widths); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) renderNode(w io.Writer, node *model.Node, prefix string, isLast bool, widths columnWidths) error {
	connector := "┣━ "
	childPrefix := prefix + "┃  "
	if isLast {
		connector = "┗━ "
		childPrefix = prefix + "   "
	}

	if _, err := fmt.Fprintln(w, prefix+connector+r.renderLine(node, widths)); err != nil {
		return err
	}
	if len(node.Preview) > 0 {
		if err := r.renderPreview(w, node, childPrefix, len(node.Children) == 0); err != nil {
			return err
		}
	}
	return r.renderChildren(w, node.Children, childPrefix)
}

func (r Renderer) renderPreview(w io.Writer, node *model.Node, prefix string, noChildren bool) error {
	for idx, line := range node.Preview {
		connector := "├─ "
		if idx == len(node.Preview)-1 && noChildren {
			connector = "└─ "
		}
		text := strings.TrimRight(line.Text, "\n")
		text = r.applyHighlights(text, line.Highlights, nil)
		if _, err := fmt.Fprintln(w, prefix+connector+text); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) renderLine(node *model.Node, widths columnWidths) string {
	var builder strings.Builder
	if r.prepared.Long {
		builder.WriteString(permissionString(node))
		builder.WriteString(" ")
		builder.WriteString(padLeft(r.sizeText(node.Size), widths.size))
		builder.WriteString(" ")
		builder.WriteString(node.ModTime.Format("2006-01-02 15:04"))
		builder.WriteString(" ")
	}
	builder.WriteString(r.renderLabel(node))
	return builder.String()
}

func (r Renderer) renderLabel(node *model.Node) string {
	icon := ""
	if r.prepared.UseIcons {
		icon = nodeIcon(node) + " "
	}

	suffix := ""
	if r.prepared.Classify {
		suffix = node.Classifier
	}

	base := r.baseColor(node)
	label := icon + r.applyHighlights(node.Name, node.NameHighlights, base)
	if suffix != "" {
		label += styleSegment(suffix, base)
	}
	return label
}

func (r Renderer) baseColor(node *model.Node) *color.Color {
	if !r.prepared.UseColor {
		return nil
	}
	if node.IsDir {
		return r.dirColor
	}
	switch strings.ToLower(filepath.Ext(node.Name)) {
	case ".go":
		return r.goColor
	case ".json", ".yaml", ".yml", ".toml":
		return r.jsonColor
	case ".sh", ".ps1", ".cmd", ".bat", ".exe", ".com":
		return r.scriptColor
	default:
		return nil
	}
}

func (r Renderer) applyHighlights(text string, ranges []model.Range, base *color.Color) string {
	if text == "" {
		return text
	}
	if len(ranges) == 0 {
		if base == nil || !r.prepared.UseColor {
			return text
		}
		return base.Sprint(text)
	}
	if !r.prepared.UseColor {
		return text
	}

	var builder strings.Builder
	cursor := 0
	for _, item := range ranges {
		if item.Start > cursor {
			builder.WriteString(styleSegment(text[cursor:item.Start], base))
		}
		if item.Start >= len(text) {
			break
		}
		end := item.End
		if end > len(text) {
			end = len(text)
		}
		builder.WriteString(r.highlightColor.Sprint(text[item.Start:end]))
		cursor = end
	}
	if cursor < len(text) {
		builder.WriteString(styleSegment(text[cursor:], base))
	}
	return builder.String()
}

func (r Renderer) measureColumns(nodes []*model.Node) columnWidths {
	widths := columnWidths{}
	if !r.prepared.Long {
		return widths
	}
	for _, node := range nodes {
		width := runeWidth(r.sizeText(node.Size))
		if width > widths.size {
			widths.size = width
		}
	}
	return widths
}

func (r Renderer) sizeText(size int64) string {
	if r.prepared.HumanReadable {
		return formatSize(size)
	}
	return strconv.FormatInt(size, 10)
}

func nodeIcon(node *model.Node) string {
	if node.IsDir {
		return "📁"
	}
	return "📄"
}

func styleSegment(text string, base *color.Color) string {
	if base == nil {
		return text
	}
	return base.Sprint(text)
}

func permissionString(node *model.Node) string {
	mode := node.Mode
	prefix := '-'
	switch {
	case node.IsSymlink:
		prefix = 'l'
	case node.IsDir:
		prefix = 'd'
	}

	bits := []struct {
		mask os.FileMode
		char byte
	}{
		{0o400, 'r'}, {0o200, 'w'}, {0o100, 'x'},
		{0o040, 'r'}, {0o020, 'w'}, {0o010, 'x'},
		{0o004, 'r'}, {0o002, 'w'}, {0o001, 'x'},
	}

	out := []byte{byte(prefix)}
	for _, bit := range bits {
		if mode&bit.mask != 0 {
			out = append(out, bit.char)
		} else {
			out = append(out, '-')
		}
	}
	return string(out)
}

func padLeft(value string, width int) string {
	pad := width - runeWidth(value)
	if pad <= 0 {
		return value
	}
	return strings.Repeat(" ", pad) + value
}

func runeWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func formatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d%s", size, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}
