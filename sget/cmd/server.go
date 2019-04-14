// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"

	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/multiformats/go-multihash"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		server()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

}

// TODO(philips: terrible hack
var directory string

type repo struct {
	directory string
	repo      git.Repository
}

func (r repo) handler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		io.WriteString(resp, "Only POST is supported!")
		return
	}

	req.ParseForm()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer(body)

	sum, err := readerDigest(buf)
	mhbuf, _ := multihash.EncodeName(sum, "sha256")
	mhhex := hex.EncodeToString(mhbuf)
	println(mhhex)

	filename := filepath.Join(directory, mhhex)
	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		panic(err)
	}

	w, err := r.repo.Worktree()
	if err != nil {
		panic(err)
	}

	_, err = w.Add(mhhex)
	if err != nil {
		panic(err)
	}

	status, err := w.Status()
	if err != nil {
		panic(err)
	}

	// Commits the current staging are to the repository, with the new file
	// just created. We should provide the object.Signature of Author of the
	// commit.
	co, err := w.Commit("example go-git commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})

	fmt.Println(status)
	obj, err := r.repo.CommitObject(co)
	if err != nil {
		panic(err)
	}

	fmt.Println(obj)

	auth := &githttp.BasicAuth{
		Username: "philips",
		Password: "00f9a4bab7616d0a6b4e1feea76eade10cfc7739",
	}

	fmt.Printf("git push\n")
	// push using default options
	err = r.repo.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		panic(err)
	}
}

func server() {
	url := os.Args[2]
	directory = os.Args[3]

	repo := repo{
		directory: directory,
	}

	// Clone the given repository to the given directory
	fmt.Printf("git clone %s %s --recursive\n", url, directory)
	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	repo.repo = *r

	if err != nil {
		panic(err)
	}

	ref, err := r.Head()
	if err != nil {
		panic(err)
	}

	_, err = r.CommitObject(ref.Hash())
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", repo.handler)
	log.Fatal(http.ListenAndServe(":5001", nil))
}
