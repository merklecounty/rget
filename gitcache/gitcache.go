package gitcache

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type GitCache struct {
	dir  autocert.DirCache
	repo git.Repository
}

func NewGitCache(url, dir string) (*GitCache, error) {
	gc := GitCache{
		dir: autocert.DirCache(dir),
	}

	auth := &githttp.BasicAuth{
		Username: "philips",
		Password: "00f9a4bab7616d0a6b4e1feea76eade10cfc7739",
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("git clone %s %s --recursive\n", url, dir)
		r, err := git.PlainClone(dir, false, &git.CloneOptions{
			Auth:              auth,
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		if err != nil {
			return nil, err
		}

		gc.repo = *r
	} else {
		r, err := git.PlainOpen(dir)
		if err != nil {
			return nil, err
		}
		gc.repo = *r

		w, err := r.Worktree()
		if err != nil {
			return nil, err
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	}

	ref, err := gc.repo.Head()
	if err != nil {
		return nil, err
	}

	_, err = gc.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	return &gc, nil
}

func (g GitCache) Delete(ctx context.Context, name string) error {
	err := g.dir.Delete(ctx, name)

	// TODO: do git stuff here

	return err
}

func (g GitCache) Get(ctx context.Context, name string) ([]byte, error) {
	return g.dir.Get(ctx, name)
}

func (g GitCache) Put(ctx context.Context, name string, data []byte) error {
	err := g.dir.Put(ctx, name, data)

	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	_, err = w.Add(name)
	if err != nil {
		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}

	// Commits the current staging are to the repository, with the new file
	// just created. We should provide the object.Signature of Author of the
	// commit.
	co, err := w.Commit(fmt.Sprintf("add: %v", name), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "sget bot",
			Email: "sget@ifup.org",
			When:  time.Now(),
		},
	})

	fmt.Println(status)
	obj, err := g.repo.CommitObject(co)
	if err != nil {
		return err
	}

	fmt.Println(obj)

	auth := &githttp.BasicAuth{
		Username: "philips",
		Password: "00f9a4bab7616d0a6b4e1feea76eade10cfc7739",
	}

	fmt.Printf("git push\n")
	// push using default options
	err = g.repo.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		return err
	}

	return nil
}
