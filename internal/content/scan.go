package content

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"mgtree/internal/model"
)

type Request struct {
	Path             string
	PreviewLines     int
	ContentExpr      *regexp.Regexp
	Deep             bool
	MaxScanBytes     int64
	PreviewReadBytes int64
}

type Result struct {
	Matched       bool
	Preview       []model.PreviewLine
	Scanned       bool
	SkippedLarge  bool
	SkippedBinary bool
}

func InspectFile(req Request) (Result, error) {
	info, err := os.Stat(req.Path)
	if err != nil {
		return Result{}, err
	}
	if !req.Deep && info.Size() > req.MaxScanBytes {
		return Result{SkippedLarge: true}, nil
	}

	data, err := os.ReadFile(req.Path)
	if err != nil {
		return Result{}, err
	}
	if isBinary(data) {
		return Result{SkippedBinary: true}, nil
	}

	text := normalizeNewlines(string(data))
	lines := splitLines(text)
	result := Result{Scanned: true}

	if req.ContentExpr != nil {
		matchIndex := req.ContentExpr.FindStringIndex(text)
		result.Matched = matchIndex != nil
		if result.Matched && req.PreviewLines > 0 {
			matchLine := lineIndexAtOffset(lines, matchIndex[0])
			result.Preview = previewAroundMatch(lines, matchLine, req.PreviewLines, req.ContentExpr)
		}
	}

	if req.PreviewLines > 0 && len(result.Preview) == 0 {
		previewText := text
		if !req.Deep && req.PreviewReadBytes > 0 {
			previewText = normalizeNewlines(string(limitBytes(data, req.PreviewReadBytes)))
		}
		result.Preview = smartPreview(splitLines(previewText), req.PreviewLines)
	}

	return result, nil
}

func normalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func splitLines(text string) []string {
	return strings.Split(text, "\n")
}

func previewAroundMatch(lines []string, matchLine, previewLines int, expr *regexp.Regexp) []model.PreviewLine {
	if previewLines <= 0 || len(lines) == 0 {
		return nil
	}
	start := matchLine - (previewLines / 2)
	if start < 0 {
		start = 0
	}
	end := start + previewLines
	if end > len(lines) {
		end = len(lines)
		start = end - previewLines
		if start < 0 {
			start = 0
		}
	}

	preview := make([]model.PreviewLine, 0, end-start)
	for idx := start; idx < end; idx++ {
		preview = append(preview, model.PreviewLine{
			Number:     idx + 1,
			Text:       lines[idx],
			Highlights: regexHighlights(lines[idx], expr),
		})
	}
	return preview
}

func smartPreview(lines []string, previewLines int) []model.PreviewLine {
	if previewLines <= 0 || len(lines) == 0 {
		return nil
	}
	start := 0
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "--") {
			continue
		}
		start = idx
		break
	}
	end := start + previewLines
	if end > len(lines) {
		end = len(lines)
	}

	preview := make([]model.PreviewLine, 0, end-start)
	for idx := start; idx < end; idx++ {
		preview = append(preview, model.PreviewLine{
			Number: idx + 1,
			Text:   lines[idx],
		})
	}
	return preview
}

func regexHighlights(text string, expr *regexp.Regexp) []model.Range {
	if expr == nil {
		return nil
	}
	indices := expr.FindAllStringIndex(text, -1)
	highlights := make([]model.Range, 0, len(indices))
	for _, idx := range indices {
		highlights = append(highlights, model.Range{Start: idx[0], End: idx[1]})
	}
	return highlights
}

func lineIndexAtOffset(lines []string, offset int) int {
	position := 0
	for idx, line := range lines {
		if offset <= position+len(line) {
			return idx
		}
		position += len(line) + 1
	}
	if len(lines) == 0 {
		return 0
	}
	return len(lines) - 1
}

func isBinary(data []byte) bool {
	sample := limitBytes(data, 8<<10)
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if !utf8.Valid(sample) {
		return true
	}
	runeSample := []rune(string(sample))
	if len(runeSample) == 0 {
		return false
	}
	printable := 0
	for _, r := range runeSample {
		if r == '\n' || r == '\r' || r == '\t' || r >= 32 {
			printable++
		}
	}
	return float64(printable)/float64(len(runeSample)) < 0.85
}

func limitBytes(data []byte, max int64) []byte {
	if max <= 0 || int64(len(data)) <= max {
		return data
	}
	return data[:max]
}
