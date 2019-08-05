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
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"go.merklecounty.com/rget/autocert"
	"go.merklecounty.com/rget/gitcache"
	"go.merklecounty.com/rget/rgethash"
	"go.merklecounty.com/rget/rgetserver"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "rget API server and TLS server",
	Long: `Provides an HTTP and HTTPS server that saves into two different
git repos one with TLS secrets and one with public data that can be audited.`,
	Run: server,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func server(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Printf("missing required arguments (public git URL, private git URL)\n")
		os.Exit(1)
	}
	pubgit := args[0]
	privgit := args[1]

	username := os.Getenv("GITHUB_USERNAME")
	password := os.Getenv("GITHUB_PASSWORD")

	if username == "" || password == "" {
		fmt.Printf("environment variables GITHUB_USERNAME and GITHUB_PASSWORD must be set\n")
		os.Exit(1)
	}

	auth := githttp.BasicAuth{
		Username: username,
		Password: password,
	}

	pubgc, err := gitcache.NewGitCache(pubgit, &auth, "public")
	if err != nil {
		panic(err)
	}
	rr := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rget_project_requests",
		Help: "Total number of requests for a particular project",
	}, []string{"method", "project"})

	rs := rgetserver.Server{
		GitCache: pubgc,
		ProjReqs: rr,
	}

	http.HandleFunc("/", rs.ReleaseHandler)
	http.HandleFunc("/api/", rs.APIHandler)

	privgc, err := gitcache.NewGitCache(privgit, &auth, "private")
	if err != nil {
		panic(err)
	}

	hostPolicy := rgethash.HostPolicyFunc(pubgc)

	hostPolicyLog := func(ctx context.Context, host string) (autocert.Policy, error) {
		policy, err := hostPolicy(ctx, host)
		fmt.Printf("hostPolicy: %v err: %v\n", policy, err)
		return policy, err
	}

	m := &autocert.Manager{
		Cache:      privgc,
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicyLog,
		Email:      "letsencrypt@merklecounty.com",
	}
	s := &http.Server{
		Addr:      ":https",
		TLSConfig: m.TLSConfig(),
	}
	go func() {
		err := s.ListenAndServeTLS("", "")
		if err != nil {
			panic(err)
		}
	}()

	ms := &http.Server{
		Addr:    ":2112",
		Handler: promhttp.Handler(),
	}

	go func() {
		err := ms.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	log.Fatal(http.ListenAndServe(":http", nil))
}
