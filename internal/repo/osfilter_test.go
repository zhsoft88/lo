package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestEntryKeyAndParse(t *testing.T) {
	tests := []struct {
		path    string
		osID    uint8
		wantKey string
	}{
		{"a.txt", 0, "a.txt"},
		{"a.txt", OSID("linux"), "a.txt\x00\x03"},
		{"path/to/file.go", OSID("win"), "path/to/file.go\x00\x01"},
		{"", 0, ""},
	}
	for _, tt := range tests {
		key := entryKey(tt.path, tt.osID)
		if key != tt.wantKey {
			t.Errorf("entryKey(%q, %d) = %q, want %q", tt.path, tt.osID, key, tt.wantKey)
		}
		gotPath, gotOS := parseKey(key)
		if gotPath != tt.path || gotOS != tt.osID {
			t.Errorf("parseKey(%q) = (%q, %d), want (%q, %d)", key, gotPath, gotOS, tt.path, tt.osID)
		}
	}
}

func TestMatchOS(t *testing.T) {
	if !matchOS(0, OSID("linux")) {
		t.Error("empty OS should match any OS")
	}
	if !matchOS(OSID("linux"), OSID("linux")) {
		t.Error("linux should match linux")
	}
	if matchOS(OSID("linux"), OSID("win")) {
		t.Error("linux should not match windows")
	}
	if !matchOS(0, 99) {
		t.Error("empty OS should match unknown OS")
	}
}

func TestVisibleEntries(t *testing.T) {
	entries := map[string]IndexEntry{
		"default.txt":                    {OS: 0},
		"default.txt\x00\x03":           {OS: OSID("linux")},
		"default.txt\x00\x01":           {OS: OSID("win")},
		"shared.txt":                     {OS: 0},
		"linux_only.txt\x00\x03":        {OS: OSID("linux")},
		"win_only.txt\x00\x01":          {OS: OSID("win")},
	}

	// On Linux
	visible := visibleEntries(entries, OSID("linux"))
	if _, ok := visible["default.txt"]; !ok {
		t.Error("expected default.txt on linux")
	}
	if _, ok := visible["shared.txt"]; !ok {
		t.Error("expected shared.txt on linux")
	}
	if _, ok := visible["linux_only.txt"]; !ok {
		t.Error("expected linux_only.txt on linux")
	}
	if _, ok := visible["win_only.txt"]; ok {
		t.Error("win_only.txt should NOT be visible on linux")
	}

	// On Linux, OS-specific should override default
	if e, ok := visible["default.txt"]; !ok || e.OS != OSID("linux") {
		t.Error("linux version of default.txt should override default on linux")
	}

	// On Windows
	visible = visibleEntries(entries, OSID("win"))
	if _, ok := visible["win_only.txt"]; !ok {
		t.Error("expected win_only.txt on windows")
	}
	if _, ok := visible["linux_only.txt"]; ok {
		t.Error("linux_only.txt should NOT be visible on windows")
	}
	if e, ok := visible["default.txt"]; !ok || e.OS != OSID("win") {
		t.Error("windows version of default.txt should override default on windows")
	}
}

func TestCollectPaths(t *testing.T) {
	entries := map[string]IndexEntry{
		"a.txt":              {},
		"a.txt\x00\x03":      {},
		"a.txt\x00\x01":      {},
		"b.txt":              {},
		"sub/c.txt":          {},
	}
	paths := collectPaths(entries)
	m := make(map[string]bool)
	for _, p := range paths {
		m[p] = true
	}
	if !m["a.txt"] {
		t.Error("expected a.txt in paths")
	}
	if !m["b.txt"] {
		t.Error("expected b.txt in paths")
	}
	if !m["sub/c.txt"] {
		t.Error("expected sub/c.txt in paths")
	}
	if len(m) != 3 {
		t.Errorf("expected 3 unique paths, got %d", len(m))
	}
}

func TestIsKnownOS(t *testing.T) {
	if !IsKnownOS("linux") {
		t.Error("expected linux to be known")
	}
	if !IsKnownOS("win") {
		t.Error("expected windows to be known")
	}
	if !IsKnownOS("mac") {
		t.Error("expected darwin to be known")
	}
	if IsKnownOS("foobar") {
		t.Error("expected foobar to be unknown")
	}
}

func TestOSIDAndName(t *testing.T) {
	if id := OSID("linux"); id != 3 {
		t.Errorf("OSID(linux) = %d, want 3", id)
	}
	if name := OSName(3); name != "linux" {
		t.Errorf("OSName(3) = %q, want %q", name, "linux")
	}
	if name := OSNameOrStar(0); name != "*" {
		t.Errorf("OSNameOrStar(0) = %q, want %q", name, "*")
	}
	if name := OSNameOrStar(OSID("win")); name != "win" {
		t.Errorf("OSNameOrStar(win) = %q, want %q", name, "win")
	}
}

// ---- Integration tests ----

func TestAddFileWithOS(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-os-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Add default file
	ioutil.WriteFile(filepath.Join(dir, "shared.txt"), []byte("shared"), 0644)
	if err := r.AddFile(filepath.Join(dir, "shared.txt")); err != nil {
		t.Fatal(err)
	}

	// Add linux-specific file
	ioutil.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	if err := r.AddFileOS(filepath.Join(dir, "main.go"), "linux"); err != nil {
		t.Fatal(err)
	}

	// Verify both exist in index
	idx, err := r.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := idx.Entries["shared.txt"]; !ok {
		t.Error("expected shared.txt in index")
	}
	if _, ok := idx.Entries["main.go\x00\x03"]; !ok {
		t.Error("expected main.go linux variant in index")
	}
	if e := idx.Entries["main.go\x00\x03"]; e.OS != OSID("linux") {
		t.Errorf("expected OS=3 (linux), got %d", e.OS)
	}
}

func TestAddDefaultAndOSSamePath(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-os-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Add default version
	ioutil.WriteFile(filepath.Join(dir, "config.yaml"), []byte("default config"), 0644)
	if err := r.AddFile(filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}

	// Add linux version
	ioutil.WriteFile(filepath.Join(dir, "config.yaml"), []byte("linux config"), 0644)
	if err := r.AddFileOS(filepath.Join(dir, "config.yaml"), "linux"); err != nil {
		t.Fatal(err)
	}

	// Verify both coexist
	idx, err := r.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Entries["config.yaml"]; !ok {
		t.Error("expected default config.yaml in index")
	}
	if _, ok := idx.Entries["config.yaml\x00\x03"]; !ok {
		t.Error("expected linux config.yaml in index")
	}

	// Build tree and verify OS field preserved
	tree, err := r.BuildTree()
	if err != nil {
		t.Fatal(err)
	}
	var foundDefault, foundLinux bool
	for _, e := range tree.Entries {
		if e.Name == "config.yaml" && e.OS == 0 {
			foundDefault = true
		}
		if e.Name == "config.yaml" && e.OS == OSID("linux") {
			foundLinux = true
		}
	}
	if !foundDefault {
		t.Error("expected default entry in tree")
	}
	if !foundLinux {
		t.Error("expected linux entry in tree")
	}

	// Commit and checkout on current OS
	h, err := r.WriteCommit("Test", "os test")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.restoreCommit(h); err != nil {
		t.Fatal(err)
	}

	// After checkout, verify the correct OS variant is in the index
	idx2, err := r.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx2.Entries["config.yaml"]; !ok {
		t.Error("expected default config.yaml in index after checkout")
	}
	if _, ok := idx2.Entries["config.yaml\x00\x03"]; !ok {
		t.Error("expected linux config.yaml in index after checkout")
	}

	// Verify the correct variant is visible on the current OS
	visible := visibleEntries(idx2.Entries, currentOS())
	e, ok := visible["config.yaml"]
	if !ok {
		t.Error("expected config.yaml in visible entries after checkout")
	} else if currentOS() == OSID("linux") && e.OS != OSID("linux") {
		t.Error("expected linux variant to be visible on linux")
	} else if currentOS() != OSID("linux") && e.OS != 0 {
		t.Error("expected default variant to be visible on non-linux OS")
	}

	// Verify working tree file exists
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); os.IsNotExist(err) {
		t.Error("expected config.yaml in working tree")
	}
}

func TestRemoveFileWithOS(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-os-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	ioutil.WriteFile(filepath.Join(dir, "f.txt"), []byte("data"), 0644)
	r.AddFile(filepath.Join(dir, "f.txt"))
	r.AddFileOS(filepath.Join(dir, "f.txt"), "linux")

	// Remove only linux variant
	if err := r.RemoveFileOS(filepath.Join(dir, "f.txt"), "linux"); err != nil {
		t.Fatal(err)
	}
	idx, _ := r.LoadIndex()
	if _, ok := idx.Entries["f.txt"]; !ok {
		t.Error("expected default f.txt to remain")
	}
	if _, ok := idx.Entries["f.txt\x00\x03"]; ok {
		t.Error("expected linux variant to be removed")
	}

	// Remove all variants
	if err := r.RemoveFile(filepath.Join(dir, "f.txt")); err != nil {
		t.Fatal(err)
	}
	idx2, _ := r.LoadIndex()
	if len(idx2.Entries) != 0 {
		t.Error("expected all entries removed")
	}
}

func TestStatusVisibleOnly(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-os-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Add files for all OSes
	ioutil.WriteFile(filepath.Join(dir, "shared.txt"), []byte("shared"), 0644)
	r.AddFile(filepath.Join(dir, "shared.txt"))

	ioutil.WriteFile(filepath.Join(dir, "os.txt"), []byte("data"), 0644)
	r.AddFileOS(filepath.Join(dir, "os.txt"), OSName(currentOS()))

	ioutil.WriteFile(filepath.Join(dir, "other.txt"), []byte("other"), 0644)
	otherOSName := "win"
	if currentOS() == OSID("win") {
		otherOSName = "linux"
	}
	r.AddFileOS(filepath.Join(dir, "other.txt"), otherOSName)

	// Status should only show current OS entries
	s, err := r.WorkTreeStatus()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := s.Staged["shared.txt"]; !ok {
		t.Error("expected shared.txt in status")
	}
	if _, ok := s.Staged["os.txt"]; !ok {
		t.Error("expected current OS file in status")
	}
	if _, ok := s.Staged["other.txt"]; ok {
		t.Error("expected other OS file NOT in status")
	}
}

func TestLazyCloneWithOS(t *testing.T) {
	sourceDir, err := ioutil.TempDir("", "lo-source-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(sourceDir)

	cloneDir, err := ioutil.TempDir("", "lo-clone-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cloneDir)

	// Source with OS-specific files
	source, err := Init(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(filepath.Join(sourceDir, "shared.txt"), []byte("hello"), 0644)
	source.AddFile(filepath.Join(sourceDir, "shared.txt"))
	ioutil.WriteFile(filepath.Join(sourceDir, "f.txt"), []byte("linux data"), 0644)
	source.AddFileOS(filepath.Join(sourceDir, "f.txt"), "linux")
	ioutil.WriteFile(filepath.Join(sourceDir, "f.txt"), []byte("windows data"), 0644)
	source.AddFileOS(filepath.Join(sourceDir, "f.txt"), "win")
	_, err = source.WriteCommit("Test", "initial")
	if err != nil {
		t.Fatal(err)
	}

	// Clone via local path
	r, err := Clone(sourceDir, cloneDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify current OS variant is checked out
	data, err := ioutil.ReadFile(filepath.Join(cloneDir, "shared.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected 'hello', got %q", data)
	}

	data, err = ioutil.ReadFile(filepath.Join(cloneDir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}

	// The content should match current OS
	expectedContent := "linux data"
	if currentOS() == OSID("win") {
		expectedContent = "windows data"
	}
	if string(data) != expectedContent {
		t.Fatalf("expected %q for current OS, got %q", expectedContent, data)
	}

	// Index should have all OS variants
	idx, err := r.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Entries["f.txt\x00\x03"]; !ok {
		t.Error("expected linux variant in index after clone")
	}
	if _, ok := idx.Entries["f.txt\x00\x01"]; !ok {
		t.Error("expected windows variant in index after clone")
	}
}

func TestShowOS(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-os-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(filepath.Join(dir, "f.txt"), []byte("default content"), 0644)
	r.AddFile(filepath.Join(dir, "f.txt"))
	ioutil.WriteFile(filepath.Join(dir, "f.txt"), []byte("linux content"), 0644)
	r.AddFileOS(filepath.Join(dir, "f.txt"), "linux")

	// Commit and checkout
	h, err := r.WriteCommit("Test", "test")
	if err != nil {
		t.Fatal(err)
	}
	r.restoreCommit(h)

	// Load the linux variant directly from index
	idx, err := r.LoadIndex()
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := idx.Entries["f.txt\x00\x03"]
	if !ok {
		t.Fatal("expected linux variant in index")
	}
	content, err := r.LoadFileContent(entry.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "linux content" {
		t.Fatalf("expected 'linux content', got %q", string(content))
	}
}
