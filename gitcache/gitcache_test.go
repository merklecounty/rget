package gitcache

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestPrefix(t *testing.T) {
	dir, err := ioutil.TempDir("", "autocert")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	url := filepath.Join(dir, "repo")
	err = os.Mkdir(url, 0755)
	if err != nil {
		t.Fatal(err)
	}

	r, err := git.PlainInit(url, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(url, "README"), []byte("Hello world"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("README")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("README\n", &git.CommitOptions{Author: &object.Signature{
		Name: "Zohra",
	}})
	if err != nil {
		t.Fatal(err)
	}

	println(url)
	gc, err := NewGitCache(filepath.Join(url, git.GitDirName), nil, filepath.Join(dir, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	expected := []string{"dummy1", "dummy1.dummy1", "dummy1.dummy1.dummy1"}
	for _, n := range expected {
		if err := gc.Put(ctx, n, []byte{1}); err != nil {
			t.Fatalf("put: %v", err)
		}
	}

	matches, err := gc.Prefix(ctx, "dummy")
	if err != nil {
		t.Fatalf("prefix: %v", err)
	}

	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("matches = %v; want %v", matches, expected)
	}
}
