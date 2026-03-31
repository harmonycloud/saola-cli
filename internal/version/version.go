package version

import "fmt"

// These variables are set via -ldflags at build time.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Info holds version information.
//
// Info 保存版本信息。
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
}

// Get returns the current version info.
//
// 返回当前版本信息。
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
	}
}

// String returns a human-readable version string.
//
// 返回人类可读的版本字符串。
func (i Info) String() string {
	return fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Date: %s", i.Version, i.GitCommit, i.BuildDate)
}
