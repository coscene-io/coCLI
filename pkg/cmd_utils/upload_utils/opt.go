package upload_utils

import (
	"os"
	"path/filepath"

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
	Recursive     bool
	IncludeHidden bool

	// Additional mapping from file path to oss path
	AdditionalUploads map[string]string
}

func (opt *FileOpts) Valid() error {
	if opt.Path == "" && len(opt.AdditionalUploads) == 0 {
		return errors.New("file path empty")
	}

	if opt.Path == "" {
		return nil
	}

	opt.relDir = opt.Path
	fileInfo, err := os.Stat(opt.Path)
	if err != nil {
		return errors.Wrap(err, "invalid file path")
	}
	if !fileInfo.IsDir() {
		opt.relDir = filepath.Dir(opt.Path)
	}
	return nil
}
