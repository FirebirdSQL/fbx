package fbconn

import (
	"fmt"
	"regexp"
	"strconv"
)

var serverVersionRegexp = regexp.MustCompile(`((\w{2})-(\w)(\d+)\.(\d+)\.(\d+)\.(\d+)(?:-\S+)?) (.+)`)

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
	for i, match := range matches[4:8] {
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

// Database info encodes a count followed by one-byte-length-prefixed version
// strings. The first string identifies the server, the rest describe the path.
func parseDatabaseServerVersion(data string) (ServerVersion, error) {
	invalid := func() (ServerVersion, error) {
		return ServerVersion{}, fmt.Errorf("invalid database version entries: %q", data)
	}
	if len(data) == 0 || data[0] == 0 {
		return invalid()
	}
	var first string
	pos := 1
	for i := 0; i < int(data[0]); i++ {
		if pos >= len(data) {
			return invalid()
		}
		length := int(data[pos])
		pos++
		if length > len(data)-pos {
			return invalid()
		}
		if i == 0 {
			first = data[pos : pos+length]
		}
		pos += length
	}
	if pos != len(data) {
		return invalid()
	}
	return parseServerVersion(first)
}
