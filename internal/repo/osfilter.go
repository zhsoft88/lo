package repo

import (
	"runtime"
)

// KnownOSes is the list of recognized operating system short identifiers.
// The numeric ID = index + 1. ID 0 means "all OSes" (default).
var KnownOSes = []string{
	"win",       // 1
	"mac",       // 2
	"linux",     // 3
	"freebsd",   // 4
	"netbsd",    // 5
	"openbsd",   // 6
	"dragonfly", // 7
	"solaris",   // 8
	"android",   // 9
}

// osNameToID maps OS name strings to numeric IDs.
var osNameToID map[string]uint8

// osIDToName maps numeric OS IDs to name strings. ID 0 is "".
var osIDToName map[uint8]string

func init() {
	osNameToID = make(map[string]uint8, len(KnownOSes))
	osIDToName = make(map[uint8]string, len(KnownOSes))
	for i, name := range KnownOSes {
		id := uint8(i + 1)
		osNameToID[name] = id
		osIDToName[id] = name
	}
}

// IsKnownOS reports whether s is a known OS identifier.
func IsKnownOS(s string) bool {
	_, ok := osNameToID[s]
	return ok
}

// OSID returns the numeric ID for an OS name string.
// Returns 0 if the name is unknown (0 = all OSes).
func OSID(name string) uint8 {
	if id, ok := osNameToID[name]; ok {
		return id
	}
	return 0
}

// OSName returns the name string for a numeric OS ID.
// Returns "" for ID 0 (all OSes), "?" for unknown IDs.
func OSName(id uint8) string {
	if id == 0 {
		return ""
	}
	if name, ok := osIDToName[id]; ok {
		return name
	}
	return "?"
}

// goosToOSID maps runtime.GOOS values to OS numeric IDs.
var goosToOSID = map[string]uint8{
	"windows":   1, // win
	"darwin":    2, // mac
	"linux":     3,
	"freebsd":   4,
	"netbsd":    5,
	"openbsd":   6,
	"dragonfly": 7,
	"solaris":   8,
	"android":   9,
}

// entryKey builds the composite map key for an OS-tagged entry.
// When osID is 0, key == path (backward compatible).
// When osID is non-zero, key == path + "\x00" + byte(osID).
func entryKey(path string, osID uint8) string {
	if osID == 0 {
		return path
	}
	return path + "\x00" + string([]byte{osID})
}

// EntryKey is the exported version of entryKey, for use by external packages.
func EntryKey(path string, osID uint8) string {
	return entryKey(path, osID)
}

// parseKey splits a composite key into the base path and OS ID.
// If no separator is found, OS ID is 0 (default entry).
func parseKey(key string) (path string, osID uint8) {
	for i := 0; i < len(key); i++ {
		if key[i] == '\x00' {
			if i+1 < len(key) {
				osID = uint8(key[i+1])
			}
			return key[:i], osID
		}
	}
	return key, 0
}

// ParseKey is the exported version of parseKey, for use by external packages.
func ParseKey(key string) (path string, osID uint8) {
	return parseKey(key)
}

// matchOS returns true if the given entryOS should be visible on the current OS.
// 0 (default) always matches. Non-zero only matches when equal.
func matchOS(entryOS, currentOS uint8) bool {
	if entryOS == 0 {
		return true
	}
	return entryOS == currentOS
}

// visibleEntries filters the full index map to only entries that should be
// visible on the given OS. Returns a map keyed by clean path (no OS suffix).
// For each base path: if an OS-specific match exists it wins; otherwise default.
func visibleEntries(entries map[string]IndexEntry, currentOS uint8) map[string]IndexEntry {
	result := make(map[string]IndexEntry)

	for key, entry := range entries {
		path, os := parseKey(key)
		if !matchOS(os, currentOS) {
			continue
		}
		// OS-specific match overrides default for the same path
		if existing, ok := result[path]; ok {
			if existing.OS == 0 && os != 0 {
				result[path] = entry
			}
			continue
		}
		result[path] = entry
	}

	return result
}

// collectPaths extracts deduplicated clean paths from a map of composite keys.
func collectPaths(entries map[string]IndexEntry) []string {
	seen := make(map[string]bool)
	var paths []string
	for key := range entries {
		path, _ := parseKey(key)
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	return paths
}

// CurrentOSID returns the numeric OS ID for the current runtime OS.
func CurrentOSID() uint8 {
	if id, ok := goosToOSID[runtime.GOOS]; ok {
		return id
	}
	return 0
}

// currentOS returns the numeric OS ID for the current runtime OS.
func currentOS() uint8 {
	return CurrentOSID()
}

// OSNameOrStar returns the display name for an OS ID.
// 0 is displayed as "*" (all OSes).
func OSNameOrStar(id uint8) string {
	if id == 0 {
		return "*"
	}
	return OSName(id)
}
