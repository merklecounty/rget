package gitcache // import "go.merklecounty.com/rget/gitcache"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"

	"go.merklecounty.com/rget/autocert"
)

type GitCache struct {
	dir  autocert.DirCache
	repo git.Repository
	auth transport.AuthMethod

	mu sync.Mutex
}

func prefix(dir autocert.DirCache, prefix string) (matches []string, err error) {
	subDirToSkip := ".git"

	err = filepath.Walk(string(dir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("failed accessing path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() && info.Name() == subDirToSkip {
			return filepath.SkipDir
		}
		if strings.HasPrefix(info.Name(), prefix) {
			matches = append(matches, info.Name())
		}
		return nil
	})
	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", dir, err)
		return nil, err
	}

	sort.Strings(matches)

	return
}

func NewGitCache(url string, auth transport.AuthMethod, dir string) (*GitCache, error) {
	gc := GitCache{
		dir:  autocert.DirCache(dir),
		auth: auth,
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("git clone %s %s --recursive\n", url, dir)
		r, err := git.PlainClone(dir, false, &git.CloneOptions{
			Auth:              gc.auth,
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
	g.mu.Lock()
	defer g.mu.Unlock()

	err := g.dir.Delete(ctx, name)
	if err != nil {
		return err
	}

	err = g.commit(ctx, name, "delete")
	if err != nil {
		return err
	}

	return err
}

func (g GitCache) Prefix(ctx context.Context, p string) ([]string, error) {
	matches, err := prefix(g.dir, p)
	return matches, err
}

func (g GitCache) Get(ctx context.Context, name string) ([]byte, error) {
	return g.dir.Get(ctx, name)
}

func (g GitCache) Put(ctx context.Context, name string, data []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	err := g.dir.Put(ctx, name, data)
	if err != nil {
		return err
	}

	err = g.commit(ctx, name, "put")
	if err != nil {
		return err
	}

	return nil
}

func (g GitCache) commit(ctx context.Context, name, verb string) error {
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
	co, err := w.Commit(fmt.Sprintf("%v: %v", verb, name), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Merkle County Recorder",
			Email: "security@merklecounty.com",
			When:  time.Now(),
		},
	})

	fmt.Println(status)
	obj, err := g.repo.CommitObject(co)
	if err != nil {
		return err
	}

	fmt.Println(obj)

	fmt.Printf("git push\n")
	// push using default options
	err = g.repo.Push(&git.PushOptions{
		Auth: g.auth,
	})
	if err != nil {
		return err
	}

	return nil
}
