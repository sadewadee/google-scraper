package runner

var (
	// Version is the current application version
	// It is injected at build time via -ldflags
	Version = "dev"

	// BuildDate is the timestamp of the build
	// It is injected at build time via -ldflags
	BuildDate = "unknown"

	// Commit is the git commit hash
	// It is injected at build time via -ldflags
	Commit = "none"
)
