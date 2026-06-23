package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTreeFromIndex(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{"a.txt", "b.txt", "c.txt"}
	for _, name := range files {
		fullPath := filepath.Join(dir, name)
		if err := ioutil.WriteFile(fullPath, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
		if err := repo.AddFile(fullPath); err != nil {
			t.Fatal(err)
		}
	}

	tree, err := repo.BuildTree()
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(tree.Entries))
	}

	// Verify sorted order
	for i := 1; i < len(tree.Entries); i++ {
		if tree.Entries[i].Name <= tree.Entries[i-1].Name {
			t.Fatal("entries not sorted")
		}
	}
}

func TestWriteAndLoadTree(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "data.txt")
	if err := ioutil.WriteFile(testFile, []byte("tree test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(testFile); err != nil {
		t.Fatal(err)
	}

	treeHash, err := repo.WriteTree()
	if err != nil {
		t.Fatal(err)
	}

	if treeHash.IsZero() {
		t.Fatal("expected non-zero tree hash")
	}

	loaded, err := repo.LoadTree(treeHash)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Name != "data.txt" {
		t.Fatalf("expected data.txt, got %s", loaded.Entries[0].Name)
	}
}

func TestBuildTreeEmptyIndex(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.BuildTree(); err == nil {
		t.Fatal("expected error for empty index")
	}
}

func TestWriteCommit(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "file.txt")
	if err := ioutil.WriteFile(testFile, []byte("commit test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(testFile); err != nil {
		t.Fatal(err)
	}

	commitHash, err := repo.WriteCommit("Test Author <test@test>", "initial commit")
	if err != nil {
		t.Fatal(err)
	}

	if commitHash.IsZero() {
		t.Fatal("expected non-zero commit hash")
	}

	loaded, err := repo.LoadCommit(commitHash)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Message != "initial commit" {
		t.Fatalf("expected 'initial commit', got '%s'", loaded.Message)
	}
	if loaded.Author != "Test Author <test@test>" {
		t.Fatalf("expected 'Test Author <test@test>', got '%s'", loaded.Author)
	}
	if loaded.Tree.IsZero() {
		t.Fatal("expected non-zero tree hash in commit")
	}
	if len(loaded.Parents) != 0 {
		t.Fatalf("expected 0 parents for first commit, got %d", len(loaded.Parents))
	}

	// Verify HEAD was updated
	resolved, err := repo.ResolveHEAD()
	if err != nil {
		t.Fatal(err)
	}
	if resolved != commitHash.String() {
		t.Fatalf("HEAD points to %s, expected %s", resolved, commitHash.String())
	}
}

func TestWriteMultipleCommits(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// First commit
	f1 := filepath.Join(dir, "a.txt")
	if err := ioutil.WriteFile(f1, []byte("file a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(f1); err != nil {
		t.Fatal(err)
	}
	h1, err := repo.WriteCommit("Author", "first")
	if err != nil {
		t.Fatal(err)
	}

	// Verify parents
	c1, _ := repo.LoadCommit(h1)
	if len(c1.Parents) != 0 {
		t.Fatalf("first commit: expected 0 parents, got %d", len(c1.Parents))
	}

	// Second commit
	f2 := filepath.Join(dir, "b.txt")
	if err := ioutil.WriteFile(f2, []byte("file b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(f2); err != nil {
		t.Fatal(err)
	}
	h2, err := repo.WriteCommit("Author", "second")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := repo.LoadCommit(h2)
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Parents) != 1 {
		t.Fatalf("second commit: expected 1 parent, got %d", len(c2.Parents))
	}
	if c2.Parents[0] != h1 {
		t.Fatalf("second commit parent should be first commit hash")
	}
}

func TestWriteCommitNothingStaged(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.WriteCommit("Author", "empty"); err == nil {
		t.Fatal("expected error when nothing staged")
	}
}
