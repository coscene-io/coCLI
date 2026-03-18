package upload_utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/coscene-io/cocli/api"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"golang.org/x/term"
)

type ApiOpts struct {
	api.SecurityTokenInterface
	api.FileInterface
}

var (
	defaultPartSize = uint64(1024 * 1024 * 128)
)

type UploadManagerOpts struct {
	Threads        int
	PartSize       string
	partSizeUint64 uint64
	NoTTY          bool // Force non-interactive mode
	TTY            bool // Force interactive mode
}

func (opt *UploadManagerOpts) Valid() error {
	if sizeUint64, err := opt.partSize(); err != nil {
		return errors.Wrap(err, "parse part size")
	} else {
		opt.partSizeUint64 = sizeUint64
		return nil
	}
}

func (opt *UploadManagerOpts) partSize() (uint64, error) {
	if len(opt.PartSize) == 0 {
		return defaultPartSize, nil
	}
	return humanize.ParseBytes(opt.PartSize)
}

// IsHeadlessEnvironment detects if we're running in a CI/headless environment
func IsHeadlessEnvironment() bool {
	// Check if stdin or stdout is not a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}

	// Check for common CI environment variables
	if os.Getenv("CI") == "true" {
		return true
	}

	// Check for dumb terminal
	if os.Getenv("TERM") == "dumb" {
		return true
	}

	return false
}

// ShouldUseInteractiveMode determines if interactive mode should be used
func (opt *UploadManagerOpts) ShouldUseInteractiveMode() bool {
	// Explicit flags take precedence
	if opt.NoTTY {
		return false
	}
	if opt.TTY {
		return true
	}

	// Auto-detect based on environment
	return !IsHeadlessEnvironment()
}

type FileOpts struct {
	Paths         []string
	relDir        string
	expandedPaths []string // Populated if single Path contains glob patterns
	Recursive     bool
	IncludeHidden bool
	TargetDir     string // Target directory in remote (e.g., "data/" to upload to data/ subdirectory)

	// Additional mapping from file path to oss path
	AdditionalUploads map[string]string
}

// GetPaths returns the list of paths to upload.
func (opt *FileOpts) GetPaths() []string {
	if len(opt.expandedPaths) > 0 {
		return opt.expandedPaths
	}
	return opt.Paths
}

// RelDir returns the base directory for computing relative paths.
func (opt *FileOpts) RelDir() string {
	return opt.relDir
}

func (opt *FileOpts) Valid() error {
	if len(opt.Paths) == 0 && len(opt.AdditionalUploads) == 0 {
		return errors.New("file path empty")
	}

	if len(opt.Paths) == 0 {
		return nil
	}

	// Single path: may be a glob pattern, directory, or file
	if len(opt.Paths) == 1 {
		path := opt.Paths[0]

		if hasGlobPattern(path) {
			matches, err := filepath.Glob(path)
			if err != nil {
				return errors.Wrap(err, "invalid glob pattern")
			}
			if len(matches) == 0 {
				return errors.New("glob pattern matched no files")
			}
			opt.expandedPaths = matches
			opt.relDir = globBaseDir(path)
			return nil
		}

		if _, err := os.Stat(path); err != nil {
			return errors.Wrap(err, "invalid file path")
		}
		opt.relDir = filepath.Dir(path)
		return nil
	}

	// Multiple paths (e.g. shell-expanded glob)
	for _, p := range opt.Paths {
		if _, err := os.Stat(p); err != nil {
			return errors.Wrapf(err, "invalid path: %s", p)
		}
	}
	opt.relDir = commonDir(opt.Paths)
	return nil
}

// hasGlobPattern checks if path contains glob wildcards.
func hasGlobPattern(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

// globBaseDir returns the directory part before the first wildcard in a glob pattern.
// Examples: "a/*" -> "a", "a/**/*.txt" -> "a", "a/b/c*.txt" -> "a/b"
func globBaseDir(pattern string) string {
	wildcardPos := strings.IndexAny(pattern, "*?[")
	if wildcardPos == -1 {
		return filepath.Dir(pattern)
	}

	beforeWildcard := pattern[:wildcardPos]

	if beforeWildcard == "" || beforeWildcard[len(beforeWildcard)-1] == filepath.Separator {
		return filepath.Clean(beforeWildcard)
	}
	return filepath.Dir(beforeWildcard)
}

// commonDir returns the deepest common ancestor directory of all paths.
func commonDir(paths []string) string {
	if len(paths) == 0 {
		return "."
	}
	dir := filepath.Dir(paths[0])
	for _, p := range paths[1:] {
		for !strings.HasPrefix(p, dir+string(filepath.Separator)) {
			parent := filepath.Dir(dir)
			if parent == dir {
				return dir
			}
			dir = parent
		}
	}
	return dir
}
