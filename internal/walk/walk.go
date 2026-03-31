package walk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"mgtree/internal/config"
	"mgtree/internal/content"
	"mgtree/internal/filter"
	"mgtree/internal/model"
)

func Build(prepared config.Prepared) (*model.Node, model.Stats, error) {
	rootInfo, err := os.Lstat(prepared.AbsoluteRoot)
	if err != nil {
		return nil, model.Stats{}, err
	}

	root := newNode(displayRootName(prepared.AbsoluteRoot), "", prepared.AbsoluteRoot, rootInfo)
	stats := model.Stats{}
	if root.IsDir {
		stats.ScannedFolders = 1
	} else {
		stats.ScannedFiles = 1
		stats.TotalSize = root.Size
		if err := enrichFileNode(root, prepared); err != nil {
			return nil, model.Stats{}, err
		}
	}

	if root.IsDir && !prepared.DirectoryOnly {
		childrenIndex := map[string][]*model.Node{}
		showHidden := prepared.All || prepared.AlmostAll

		err = filepath.WalkDir(prepared.AbsoluteRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == prepared.AbsoluteRoot {
				return nil
			}

			relPath, err := filepath.Rel(prepared.AbsoluteRoot, path)
			if err != nil {
				return err
			}
			relPath = filter.NormalizePath(relPath)
			depth := 1
			if relPath != "" {
				depth = strings.Count(relPath, "/") + 1
			}
			if prepared.Depth >= 0 && depth > prepared.Depth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !showHidden && isHidden(d.Name()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if filter.MatchesExclude(prepared.ExcludeRegex, relPath, d.Name()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}
			node := newNode(d.Name(), relPath, path, info)

			if node.IsDir {
				stats.ScannedFolders++
			} else {
				stats.ScannedFiles++
				stats.TotalSize += info.Size()
				if err := enrichFileNode(node, prepared); err != nil {
					return err
				}
			}

			parentKey := filter.NormalizePath(filepath.Dir(relPath))
			childrenIndex[parentKey] = append(childrenIndex[parentKey], node)
			return nil
		})
		if err != nil {
			return nil, model.Stats{}, err
		}

		attachChildren(root, childrenIndex, prepared)
	}

	finalizeTree(root, prepared, &stats, true)
	return root, stats, nil
}

func newNode(name, relPath, absPath string, info os.FileInfo) *model.Node {
	node := &model.Node{
		Name:         name,
		RelativePath: relPath,
		AbsolutePath: absPath,
		IsDir:        info.IsDir(),
		IsSymlink:    info.Mode()&os.ModeSymlink != 0,
		Size:         info.Size(),
		Mode:         info.Mode(),
		ModTime:      info.ModTime(),
		Hidden:       isHidden(name),
	}
	node.Executable = isExecutable(node.Name, node.Mode, node.IsDir, node.IsSymlink)
	node.Classifier = classifierFor(node)
	return node
}

func enrichFileNode(node *model.Node, prepared config.Prepared) error {
	if !prepared.ContentScanActive && !prepared.PreviewActive {
		return nil
	}

	result, err := content.InspectFile(content.Request{
		Path:             node.AbsolutePath,
		PreviewLines:     prepared.PreviewLines,
		ContentExpr:      prepared.ContentRegex,
		Deep:             prepared.Deep,
		MaxScanBytes:     prepared.MaxScanBytes,
		PreviewReadBytes: prepared.PreviewReadBytes,
	})
	if err != nil {
		return fmt.Errorf("inspect %s: %w", node.AbsolutePath, err)
	}

	node.ContentMatched = result.Matched
	node.ContentScanned = result.Scanned
	node.SkippedBinary = result.SkippedBinary
	node.SkippedLarge = result.SkippedLarge
	node.Preview = result.Preview
	return nil
}

func attachChildren(parent *model.Node, childrenIndex map[string][]*model.Node, prepared config.Prepared) {
	children := childrenIndex[parent.RelativePath]
	if len(children) == 0 {
		return
	}
	filter.SortChildren(children, prepared.SortMode, prepared.ReverseSort)
	parent.Children = children
	for _, child := range children {
		attachChildren(child, childrenIndex, prepared)
	}
}

func finalizeTree(node *model.Node, prepared config.Prepared, stats *model.Stats, isRoot bool) bool {
	keptChildren := make([]*model.Node, 0, len(node.Children))
	for _, child := range node.Children {
		if finalizeTree(child, prepared, stats, false) {
			keptChildren = append(keptChildren, child)
		}
	}
	node.Children = keptChildren

	eval := filter.Evaluate(prepared, node.Name, node.RelativePath, node.ContentMatched)
	node.SelfMatched = eval.SelfMatched
	node.NameMatched = eval.NameMatched
	node.SearchSatisfied = eval.SearchSatisfied
	node.NameHighlights = eval.NameHighlights

	if isRoot {
		node.Matched = true
	} else if node.IsDir {
		node.Matched = node.SelfMatched || len(node.Children) > 0
	} else {
		node.Matched = node.SelfMatched
	}

	if node.Matched {
		if node.IsDir {
			stats.MatchedFolders++
		} else {
			stats.MatchedFiles++
		}
	}
	return node.Matched
}

func displayRootName(path string) string {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return clean
	}
	return base
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func isExecutable(name string, mode os.FileMode, isDir, isSymlink bool) bool {
	if isDir || isSymlink {
		return false
	}
	if mode&0o111 != 0 {
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".bat", ".cmd", ".ps1", ".com", ".sh":
		return true
	default:
		return false
	}
}

func classifierFor(node *model.Node) string {
	switch {
	case node.IsSymlink:
		return "@"
	case node.IsDir:
		return "/"
	case node.Executable:
		return "*"
	default:
		return ""
	}
}
