package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version for a contract.
// Contract versions follow semver: MAJOR.MINOR.PATCH.
// The bump level drives the required approval process:
// PATCH = no approvals required, MINOR = team lead, MAJOR = all affected owners.
type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("version %q must be MAJOR.MINOR.PATCH", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version in %q", s)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version in %q", s)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version in %q", s)
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// BumpLevel returns what level of version bump a proposed version
// represents relative to the current version.
// Returns an error if the proposed version is not a single-level increment.
// We enforce single-level bumps to prevent version skipping, which would
// create gaps in the approval audit trail.
func BumpLevel(current, proposed Version) (string, error) {
	if proposed.Major == current.Major+1 && proposed.Minor == 0 && proposed.Patch == 0 {
		return "MAJOR", nil
	}
	if proposed.Major == current.Major && proposed.Minor == current.Minor+1 && proposed.Patch == 0 {
		return "MINOR", nil
	}
	if proposed.Major == current.Major && proposed.Minor == current.Minor && proposed.Patch == current.Patch+1 {
		return "PATCH", nil
	}
	return "", fmt.Errorf(
		"version %s is not a valid single-level increment from %s",
		proposed, current,
	)
}
