package utils

import "fmt"

// FormatBytes formats bytes using 1024-based units (KB, MB, GB, TB)
// to match the web frontend display convention.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	val := float64(b) / float64(div)
	return fmt.Sprintf("%.2f %s", val, []string{"KB", "MB", "GB", "TB", "PB"}[exp])
}
