package testutil

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func EmptyGitRepo(t *testing.T, url string) string {
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

	return filepath.Join(url, git.GitDirName)
}
