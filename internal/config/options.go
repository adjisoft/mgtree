package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	DefaultRootPath     = "."
	DefaultDepth        = -1
	DefaultMaxScanBytes = int64(1 << 20)
	DefaultPreviewBytes = int64(64 << 10)
)

type SortMode string

const (
	SortDefault   SortMode = "default"
	SortTime      SortMode = "time"
	SortSize      SortMode = "size"
	SortExtension SortMode = "extension"
	SortUnsorted  SortMode = "unsorted"
)

type Options struct {
	Root           string
	RegexPattern   string
	ExcludePattern string
	Search         string
	ContentPattern string
	ForceIcons     bool
	NoIcons        bool
	PreviewLines   int
	Depth          int
	ShowStats      bool
	ForceColor     bool
	NoColor        bool
	Fast           bool
	Deep           bool
	Long           bool
	All            bool
	AlmostAll      bool
	HumanReadable  bool
	Recursive      bool
	SortTime       bool
	SortSize       bool
	SortExtension  bool
	Unsorted       bool
	ReverseSort    bool
	OnePerLine     bool
	DirectoryOnly  bool
	Classify       bool
}

type Prepared struct {
	Options
	AbsoluteRoot      string
	IncludeRegex      *regexp.Regexp
	ExcludeRegex      *regexp.Regexp
	ContentRegex      *regexp.Regexp
	SearchFolded      string
	UseColor          bool
	UseIcons          bool
	MaxScanBytes      int64
	PreviewReadBytes  int64
	FiltersEnabled    bool
	ContentScanActive bool
	PreviewActive     bool
	SortMode          SortMode
}

func DefaultOptions() Options {
	return Options{
		Root:         DefaultRootPath,
		PreviewLines: 0,
		Depth:        DefaultDepth,
	}
}

func Prepare(opts Options, env map[string]string, isTTY bool, goos string, sortMode SortMode) (Prepared, error) {
	if opts.Root == "" {
		opts.Root = DefaultRootPath
	}
	if opts.Depth < -1 {
		return Prepared{}, errors.New("depth must be -1 or greater")
	}
	if opts.PreviewLines < 0 {
		return Prepared{}, errors.New("preview must be 0 or greater")
	}
	if opts.Fast && opts.ContentPattern != "" {
		return Prepared{}, errors.New("--fast cannot be combined with --content")
	}
	if opts.Fast && opts.PreviewLines > 0 {
		return Prepared{}, errors.New("--fast cannot be combined with --preview")
	}
	if opts.ForceColor && opts.NoColor {
		return Prepared{}, errors.New("--color cannot be combined with --no-color")
	}
	if opts.ForceIcons && opts.NoIcons {
		return Prepared{}, errors.New("--icons cannot be combined with --no-icons")
	}

	include, err := compilePattern(opts.RegexPattern, "--regex")
	if err != nil {
		return Prepared{}, err
	}
	exclude, err := compilePattern(opts.ExcludePattern, "--exclude")
	if err != nil {
		return Prepared{}, err
	}
	content, err := compilePattern(opts.ContentPattern, "--content")
	if err != nil {
		return Prepared{}, err
	}

	absoluteRoot, err := filepath.Abs(opts.Root)
	if err != nil {
		return Prepared{}, fmt.Errorf("resolve root path: %w", err)
	}

	if goos == "" {
		goos = runtime.GOOS
	}
	if sortMode == "" {
		sortMode = SortDefault
	}

	prepared := Prepared{
		Options:           opts,
		AbsoluteRoot:      absoluteRoot,
		IncludeRegex:      include,
		ExcludeRegex:      exclude,
		ContentRegex:      content,
		SearchFolded:      strings.ToLower(opts.Search),
		UseColor:          ResolveColorEnabled(opts, env, isTTY),
		UseIcons:          ResolveIconsEnabled(opts, env, isTTY, goos),
		MaxScanBytes:      DefaultMaxScanBytes,
		PreviewReadBytes:  DefaultPreviewBytes,
		ContentScanActive: content != nil,
		PreviewActive:     opts.PreviewLines > 0,
		SortMode:          sortMode,
	}
	prepared.FiltersEnabled = prepared.IncludeRegex != nil || prepared.SearchFolded != "" || prepared.ContentRegex != nil
	return prepared, nil
}

func compilePattern(pattern string, flagName string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid %s regex: %w", flagName, err)
	}
	return compiled, nil
}

func EnvMap(values []string) map[string]string {
	env := make(map[string]string, len(values))
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}

func IsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func ResolveColorEnabled(opts Options, env map[string]string, isTTY bool) bool {
	if opts.NoColor {
		return false
	}
	if opts.ForceColor {
		return true
	}
	if env["NO_COLOR"] != "" {
		return false
	}
	if env["CLICOLOR_FORCE"] != "" && env["CLICOLOR_FORCE"] != "0" {
		return true
	}
	if !isTTY {
		return false
	}
	term := strings.ToLower(env["TERM"])
	return term != "dumb"
}

func ResolveIconsEnabled(opts Options, env map[string]string, isTTY bool, goos string) bool {
	if opts.NoIcons {
		return false
	}
	if opts.ForceIcons {
		return true
	}
	if !isTTY {
		return false
	}
	if goos == "windows" {
		term := strings.ToLower(env["TERM"])
		return env["WT_SESSION"] != "" || env["TERM_PROGRAM"] != "" || strings.Contains(term, "xterm") || strings.Contains(term, "utf")
	}
	joined := strings.ToUpper(env["LANG"] + " " + env["LC_ALL"] + " " + env["LC_CTYPE"])
	return strings.Contains(joined, "UTF-8") || strings.Contains(joined, "UTF8")
}
