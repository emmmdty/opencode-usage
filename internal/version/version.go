package version

import "fmt"

var (
	Version = "0.1.0"
	Commit  = "none"
	Date    = "unknown"
)

func GetVersionInfo() string {
	return fmt.Sprintf("opencode-usage %s (commit: %s, built: %s)", Version, Commit, Date)
}
