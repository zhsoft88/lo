package repo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckout(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create and commit a file
	testFile := filepath.Join(dir, "hello.txt")
	if err := ioutil.WriteFile(testFile, []byte("checkout test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(testFile); err != nil {
		t.Fatal(err)
	}
	commitHash, err := repo.WriteCommit("Test", "first")
	if err != nil {
		t.Fatal(err)
	}

	// Remove the file and clear index
	os.Remove(testFile)
	repo.RemoveFile("hello.txt")

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatal("file should have been deleted")
	}

	// Checkout the commit
	if err := repo.Checkout(commitHash); err != nil {
		t.Fatal(err)
	}

	// Verify file is restored
	data, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "checkout test" {
		t.Fatalf("expected 'checkout test', got '%s'", data)
	}

	// Verify index is updated
	files, err := repo.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["hello.txt"]; !ok {
		t.Fatal("expected hello.txt in index after checkout")
	}
}

func TestCheckoutChunkedFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a file that will be chunked
	data := make([]byte, 10000)
	for i := range data {
		data[i] = byte(i)
	}
	testFile := filepath.Join(dir, "large.bin")
	if err := ioutil.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddFile(testFile); err != nil {
		t.Fatal(err)
	}
	commitHash, err := repo.WriteCommit("Test", "chunked file")
	if err != nil {
		t.Fatal(err)
	}

	// Remove file and checkout
	os.Remove(testFile)
	if err := repo.Checkout(commitHash); err != nil {
		t.Fatal(err)
	}

	restored, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != len(data) {
		t.Fatalf("size mismatch: %d vs %d", len(restored), len(data))
	}
	for i := range data {
		if restored[i] != data[i] {
			t.Fatalf("byte %d mismatch: %d vs %d", i, restored[i], data[i])
		}
	}
}

func TestResolveRefFullHash(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "a.txt")
	ioutil.WriteFile(testFile, []byte("a"), 0644)
	repo.AddFile(testFile)
	h, err := repo.WriteCommit("Test", "msg")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := repo.ResolveRef(h.String())
	if err != nil {
		t.Fatal(err)
	}
	if resolved != h {
		t.Fatalf("expected %s, got %s", h, resolved)
	}
}

func TestResolveRefHEAD(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "a.txt")
	ioutil.WriteFile(testFile, []byte("a"), 0644)
	repo.AddFile(testFile)
	h, err := repo.WriteCommit("Test", "msg")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := repo.ResolveRef("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != h {
		t.Fatalf("expected %s, got %s", h, resolved)
	}
}

func TestResolveRefBranch(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "a.txt")
	ioutil.WriteFile(testFile, []byte("a"), 0644)
	repo.AddFile(testFile)
	h, err := repo.WriteCommit("Test", "msg")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := repo.ResolveRef("main")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != h {
		t.Fatalf("expected %s, got %s", h, resolved)
	}
}

func TestResolveRefShortHash(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "a.txt")
	ioutil.WriteFile(testFile, []byte("a"), 0644)
	repo.AddFile(testFile)
	h, err := repo.WriteCommit("Test", "msg")
	if err != nil {
		t.Fatal(err)
	}

	short := h.Short() // first 16 hex chars
	resolved, err := repo.ResolveRef(short)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != h {
		t.Fatalf("expected %s, got %s", h, resolved)
	}
}

func TestResolveRefNotFound(t *testing.T) {
	dir, err := ioutil.TempDir("", "lo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.ResolveRef("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent ref")
	}
}
