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
	Path          string
	relDir        string
	expandedPaths []string // Populated if Path contains glob patterns
	Recursive     bool
	IncludeHidden bool
	TargetDir     string // Target directory in remote (e.g., "data/" to upload to data/ subdirectory)

	// Additional mapping from file path to oss path
	AdditionalUploads map[string]string
}

// GetPaths returns the list of paths to upload. If glob was used, returns expanded paths; otherwise returns single Path.
func (opt *FileOpts) GetPaths() []string {
	if len(opt.expandedPaths) > 0 {
		return opt.expandedPaths
	}
	if opt.Path != "" {
		return []string{opt.Path}
	}
	return nil
}

// RelDir returns the base directory for computing relative paths.
func (opt *FileOpts) RelDir() string {
	return opt.relDir
}

func (opt *FileOpts) Valid() error {
	if opt.Path == "" && len(opt.AdditionalUploads) == 0 {
		return errors.New("file path empty")
	}

	if opt.Path == "" {
		return nil
	}

	// Detect glob pattern
	if hasGlobPattern(opt.Path) {
		matches, err := filepath.Glob(opt.Path)
		if err != nil {
			return errors.Wrap(err, "invalid glob pattern")
		}
		if len(matches) == 0 {
			return errors.New("glob pattern matched no files")
		}

		opt.expandedPaths = matches
		// relDir is the base directory before the first wildcard
		opt.relDir = globBaseDir(opt.Path)
		return nil
	}

	// Regular path (no glob)
	_, err := os.Stat(opt.Path)
	if err != nil {
		return errors.Wrap(err, "invalid file path")
	}
	// Always use parent directory as relative base to preserve directory/file names
	opt.relDir = filepath.Dir(opt.Path)
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

	// If it ends with a separator, remove it; otherwise get the directory
	beforeWildcard = strings.TrimRight(beforeWildcard, string(filepath.Separator))
	return filepath.Dir(beforeWildcard)
}
