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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/philips/sget/sgethash"
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

	serverCmd.PersistentFlags().String("kubeconfig", "", "kubeconfig")
	viper.BindFlag("kubeconfig", githubCmd.PersistentFlags().Lookup("kubeconfig"))
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

	// TODO: get a URL field from the POST
	var sumsURL string
	panic("no sumsURL wired up")

	// TODO(philips): how do we ensure we don't DDoS these URLs?

	// Step 1: Download the SHA256SUMS that is correct for the URL
	response, err := http.Get(sumsURL)
	var sha256file []byte
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		var err error
		defer response.Body.Close()
		sha256file, err = ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
	}

	l := sgethash.FromSHA256SumFile(string(sha256file))

	// Step 2: Save the file contents to the git repo by domain
	domain := l.Domain()
	filename := filepath.Join(directory, domain)
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return // already have this file
	}

	err = ioutil.WriteFile(filename, body, 0644)
	if err != nil {
		panic(err)
	}

	// Step 3. Create the Certificate object for the domain and save that as well
	domain, err = sgetwellknown.Domain(sumsURL)
	if err != nil {
		fmt.Printf("wellknown domain error: %v", err)
		os.Exit(1)
	}

	sums := sgethash.FromSHA256SumFile(string(sha256file))
	ctdomain := sums.Domain() + "." + domain

	w, err := r.repo.Worktree()
	if err != nil {
		panic(err)
	}

	_, err = w.Add(domain)
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
	co, err := w.Commit(fmt.Sprintf("add: %v", domain), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "sget bot",
			Email: "sget@ifup.org",
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

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		fmt.Printf("git clone %s %s --recursive\n", url, directory)
		r, err := git.PlainClone(directory, false, &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		if err != nil {
			panic(err)
		}

		repo.repo = *r
	} else {
		r, err := git.PlainOpen(directory)
		if err != nil {
			panic(err)
		}
		repo.repo = *r

		w, err := r.Worktree()
		if err != nil {
			panic(err)
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	}

	ref, err := repo.repo.Head()
	if err != nil {
		panic(err)
	}

	_, err = repo.repo.CommitObject(ref.Hash())
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", repo.handler)
	log.Fatal(http.ListenAndServe(":5001", nil))
}
