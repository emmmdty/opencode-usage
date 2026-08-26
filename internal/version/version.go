package version

import "fmt"

var (
	Version = "0.2.1"
	Commit  = "none"
	Date    = "unknown"
)

func GetVersionInfo() string {
	return fmt.Sprintf("opencode-usage %s (commit: %s, built: %s)", Version, Commit, Date)
}
