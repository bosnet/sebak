package version

import "fmt"

var (
	Version             string // VERSION should be updated by hand at each release. It must follow SemVer (https://semver.org)
	GitCommit, GitState string // GitCommit will be overwritten automatically by the build system
	BuildDate           string // BuildDate will be overwritten automatically by the build system
)

func ToDetailVersion() string {
	return fmt.Sprintf("version=%s git=%s build=%s", Version, GitCommit, BuildDate)
}
