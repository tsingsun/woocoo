package telemetry

// Version is the current release version of the woocoo instrumentation.
func Version() string {
	return "0.58.0"
}

// SemVersion is the semantic version to be supplied to tracer/meter creation.
func SemVersion() string {
	return "semver:" + Version()
}
