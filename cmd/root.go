package cmd

import (
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"mgtree/internal/config"
	"mgtree/internal/render"
	"mgtree/internal/walk"
)

func Execute() error {
	normalizedArgs := normalizeArgs(os.Args[1:])
	root := NewRootCommand(normalizedArgs)
	root.SetArgs(normalizedArgs)
	return root.Execute()
}

func NewRootCommand(args []string) *cobra.Command {
	opts := config.DefaultOptions()
	helpRequested := false

	cmd := &cobra.Command{
		Use:          "mgtree [path]",
		Short:        "@adjisoft\nAdvanced cross-platform tree viewer with filtering, search, colors, previews, and ls-style compatibility",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, commandArgs []string) error {
			if len(commandArgs) > 0 {
				opts.Root = commandArgs[0]
			}
			if helpRequested {
				return cmd.Help()
			}

			prepared, err := config.Prepare(
				opts,
				config.EnvMap(os.Environ()),
				config.IsTerminal(os.Stdout),
				runtime.GOOS,
				detectSortMode(args),
			)
			if err != nil {
				return err
			}

			tree, stats, err := walk.Build(prepared)
			if err != nil {
				return err
			}

			renderer := render.New(prepared)
			return renderer.Render(cmd.OutOrStdout(), tree, stats)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&helpRequested, "help", "?", false, "help for mgtree")

	flags.StringVarP(&opts.RegexPattern, "regex", "e", "", "include regex for file or folder path")
	flags.StringVarP(&opts.ExcludePattern, "exclude", "x", "", "exclude regex for file or folder path")
	flags.StringVarP(&opts.Search, "search", "s", "", "search keyword in file or folder names")
	flags.StringVarP(&opts.ContentPattern, "content", "c", "", "search file contents with regex")
	flags.BoolVarP(&opts.ForceIcons, "icons", "i", false, "force icons on")
	flags.BoolVar(&opts.NoIcons, "no-icons", false, "force icons off")
	flags.IntVarP(&opts.PreviewLines, "preview", "p", 0, "show N preview lines")
	flags.IntVarP(&opts.Depth, "depth", "L", -1, "limit traversal depth")
	flags.BoolVar(&opts.ShowStats, "stats", false, "show scan and match statistics")
	flags.BoolVar(&opts.ForceColor, "color", false, "force ANSI colors on")
	flags.BoolVar(&opts.NoColor, "no-color", false, "disable colors")
	flags.BoolVar(&opts.Fast, "fast", false, "fast mode: disable content scan and preview")
	flags.BoolVar(&opts.Deep, "deep", false, "scan full file contents and ignore safe size limit")

	flags.BoolVarP(&opts.Long, "long", "l", false, "show long listing metadata")
	flags.BoolVarP(&opts.All, "all", "a", false, "show dotfiles and dot directories")
	flags.BoolVarP(&opts.AlmostAll, "almost-all", "A", false, "show dotfiles and dot directories")
	flags.BoolVarP(&opts.HumanReadable, "human-readable", "h", false, "print sizes in human-readable units")
	flags.BoolVarP(&opts.Recursive, "recursive", "R", false, "accepted for ls compatibility; mgtree is already recursive")
	flags.BoolVarP(&opts.SortTime, "sort-time", "t", false, "sort by modification time")
	flags.BoolVarP(&opts.SortSize, "sort-size", "S", false, "sort by file size")
	flags.BoolVarP(&opts.SortExtension, "sort-extension", "X", false, "sort by file extension")
	flags.BoolVarP(&opts.Unsorted, "unsorted", "U", false, "keep discovery order")
	flags.BoolVarP(&opts.ReverseSort, "reverse", "r", false, "reverse the final sort order")
	flags.BoolVarP(&opts.OnePerLine, "one-per-line", "1", false, "accepted for ls compatibility; output is already one item per line")
	flags.BoolVarP(&opts.DirectoryOnly, "directory", "d", false, "show the target itself without descending into its contents")
	flags.BoolVarP(&opts.Classify, "classify", "F", false, "append ls-style classifier suffixes to names")

	return cmd
}

func normalizeArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "-IC":
			normalized = append(normalized, "--icons")
		default:
			normalized = append(normalized, arg)
		}
	}
	return normalized
}

func detectSortMode(args []string) config.SortMode {
	mode := config.SortDefault
	longValueFlags := map[string]bool{
		"--regex":   true,
		"--exclude": true,
		"--search":  true,
		"--content": true,
		"--preview": true,
		"--depth":   true,
	}
	shortValueFlags := map[rune]bool{
		'e': true,
		'x': true,
		's': true,
		'c': true,
		'p': true,
		'L': true,
	}

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if arg == "--" {
			break
		}
		switch {
		case strings.HasPrefix(arg, "--"):
			name := strings.SplitN(arg, "=", 2)[0]
			switch name {
			case "--sort-time":
				mode = config.SortTime
			case "--sort-size":
				mode = config.SortSize
			case "--sort-extension":
				mode = config.SortExtension
			case "--unsorted":
				mode = config.SortUnsorted
			}
			if longValueFlags[name] && !strings.Contains(arg, "=") && idx+1 < len(args) {
				idx++
			}
		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			shorts := []rune(arg[1:])
			consumeNext := false
			for pos, short := range shorts {
				switch short {
				case 't':
					mode = config.SortTime
				case 'S':
					mode = config.SortSize
				case 'X':
					mode = config.SortExtension
				case 'U':
					mode = config.SortUnsorted
				}
				if shortValueFlags[short] {
					if pos == len(shorts)-1 {
						consumeNext = true
					}
					break
				}
			}
			if consumeNext && idx+1 < len(args) {
				idx++
			}
		}
	}
	return mode
}
