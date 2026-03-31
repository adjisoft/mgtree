package model

import (
	"os"
	"time"
)

type Range struct {
	Start int
	End   int
}

type PreviewLine struct {
	Number     int
	Text       string
	Highlights []Range
}

type Node struct {
	Name            string
	RelativePath    string
	AbsolutePath    string
	IsDir           bool
	IsSymlink       bool
	Size            int64
	Mode            os.FileMode
	ModTime         time.Time
	Hidden          bool
	Executable      bool
	Classifier      string
	Children        []*Node
	Preview         []PreviewLine
	NameHighlights  []Range
	SelfMatched     bool
	Matched         bool
	NameMatched     bool
	ContentMatched  bool
	ContentScanned  bool
	SkippedBinary   bool
	SkippedLarge    bool
	SearchSatisfied bool
}

type Stats struct {
	ScannedFiles   int
	ScannedFolders int
	MatchedFiles   int
	MatchedFolders int
	TotalSize      int64
}
