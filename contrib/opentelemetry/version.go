package opentelemetry

import "github.com/tsingsun/woocoo/internal"

// Version is the current release version of the woocoo instrumentation.
func Version() string {
	return internal.Version
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
func SemVersion() string {
	return "semver:" + Version()
}
