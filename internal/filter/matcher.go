package filter

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mgtree/internal/config"
	"mgtree/internal/model"
)

type Evaluation struct {
	SelfMatched     bool
	NameMatched     bool
	SearchSatisfied bool
	ContentMatched  bool
	NameHighlights  []model.Range
}

func NormalizePath(path string) string {
	normalized := filepath.ToSlash(path)
	if normalized == "." {
		return ""
	}
	return normalized
}

func MatchesExclude(expr *regexp.Regexp, relPath, name string) bool {
	if expr == nil {
		return false
	}
	return expr.MatchString(relPath) || expr.MatchString(name)
}

func Evaluate(prepared config.Prepared, name, relPath string, contentMatched bool) Evaluation {
	nameMatched := false
	searchSatisfied := true
	var highlights []model.Range

	if prepared.SearchFolded != "" {
		nameMatched = containsFolded(name, prepared.SearchFolded) || containsFolded(relPath, prepared.SearchFolded)
		searchSatisfied = nameMatched
		highlights = append(highlights, substringRanges(name, prepared.SearchFolded)...)
	}

	if prepared.ContentRegex != nil {
		searchSatisfied = contentMatched || nameMatched
	}

	includeMatched := true
	if prepared.IncludeRegex != nil {
		includeMatched = prepared.IncludeRegex.MatchString(relPath) || prepared.IncludeRegex.MatchString(name)
		highlights = append(highlights, regexRanges(name, prepared.IncludeRegex)...)
	}

	highlights = mergeRanges(highlights)

	return Evaluation{
		SelfMatched:     includeMatched && searchSatisfied,
		NameMatched:     nameMatched || len(highlights) > 0,
		SearchSatisfied: searchSatisfied,
		ContentMatched:  contentMatched,
		NameHighlights:  highlights,
	}
}

func SortChildren(nodes []*model.Node, mode config.SortMode, reverse bool) {
	if mode != config.SortUnsorted {
		sort.SliceStable(nodes, func(i, j int) bool {
			return lessNode(nodes[i], nodes[j], mode)
		})
	}
	if reverse {
		for left, right := 0, len(nodes)-1; left < right; left, right = left+1, right-1 {
			nodes[left], nodes[right] = nodes[right], nodes[left]
		}
	}
}

func lessNode(left, right *model.Node, mode config.SortMode) bool {
	leftName := strings.ToLower(left.Name)
	rightName := strings.ToLower(right.Name)

	switch mode {
	case config.SortTime:
		if !left.ModTime.Equal(right.ModTime) {
			return left.ModTime.After(right.ModTime)
		}
	case config.SortSize:
		if left.Size != right.Size {
			return left.Size > right.Size
		}
	case config.SortExtension:
		leftExt := strings.ToLower(filepath.Ext(left.Name))
		rightExt := strings.ToLower(filepath.Ext(right.Name))
		if leftExt != rightExt {
			return leftExt < rightExt
		}
	default:
		if left.IsDir != right.IsDir {
			return left.IsDir
		}
	}

	if leftName != rightName {
		return leftName < rightName
	}
	return left.AbsolutePath < right.AbsolutePath
}

func containsFolded(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), needle)
}

func substringRanges(text, foldedNeedle string) []model.Range {
	if foldedNeedle == "" {
		return nil
	}
	lower := strings.ToLower(text)
	var ranges []model.Range
	searchFrom := 0
	for {
		idx := strings.Index(lower[searchFrom:], foldedNeedle)
		if idx < 0 {
			break
		}
		start := searchFrom + idx
		ranges = append(ranges, model.Range{Start: start, End: start + len(foldedNeedle)})
		searchFrom = start + len(foldedNeedle)
		if searchFrom >= len(lower) {
			break
		}
	}
	return ranges
}

func regexRanges(text string, expr *regexp.Regexp) []model.Range {
	if expr == nil {
		return nil
	}
	indices := expr.FindAllStringIndex(text, -1)
	ranges := make([]model.Range, 0, len(indices))
	for _, idx := range indices {
		ranges = append(ranges, model.Range{Start: idx[0], End: idx[1]})
	}
	return ranges
}

func mergeRanges(ranges []model.Range) []model.Range {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Start == ranges[j].Start {
			return ranges[i].End < ranges[j].End
		}
		return ranges[i].Start < ranges[j].Start
	})
	merged := []model.Range{ranges[0]}
	for _, current := range ranges[1:] {
		last := &merged[len(merged)-1]
		if current.Start <= last.End {
			if current.End > last.End {
				last.End = current.End
			}
			continue
		}
		merged = append(merged, current)
	}
	return merged
}
