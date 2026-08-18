package fbconn

import (
	"fmt"
	"regexp"
	"strconv"
)

var serverVersionRegexp = regexp.MustCompile(`-V(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?`)

// ServerVersion identifies the Firebird or RedDatabase server version.
//
// Major, Minor, Patch, and Build contain the numeric components advertised by
// the server.
type ServerVersion struct {
	Major int
	Minor int
	Patch int
	Build int
}

// String returns the server version in major.minor.patch.build form.
func (v ServerVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Build)
}

// Compare compares v with a major.minor.patch version. It returns -1, 0, or 1
// when v is older than, equal to, or newer than the supplied version.
func (v ServerVersion) Compare(major, minor, patch int) int {
	switch {
	case v.Major != major:
		return compareInt(v.Major, major)
	case v.Minor != minor:
		return compareInt(v.Minor, minor)
	default:
		return compareInt(v.Patch, patch)
	}
}

// AtLeast reports whether v is greater than or equal to major.minor.patch.
func (v ServerVersion) AtLeast(major, minor, patch int) bool {
	return v.Compare(major, minor, patch) >= 0
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func parseServerVersion(raw string) (ServerVersion, error) {
	matches := serverVersionRegexp.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return ServerVersion{}, fmt.Errorf("server version string parsing error: %q", raw)
	}

	parts := [4]int{}
	for i, match := range matches[1:] {
		if match == "" {
			continue
		}
		part, err := strconv.Atoi(match)
		if err != nil {
			return ServerVersion{}, fmt.Errorf("server version string parsing error: %q: %w", raw, err)
		}
		parts[i] = part
	}

	return ServerVersion{
		Major: parts[0],
		Minor: parts[1],
		Patch: parts[2],
		Build: parts[3],
	}, nil
}
