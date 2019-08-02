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
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/merklecounty/rget/autocert"
	"github.com/merklecounty/rget/gitcache"
	"github.com/merklecounty/rget/rgethash"
	"github.com/merklecounty/rget/rgetwellknown"
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

type rgetServer struct {
	*gitcache.GitCache
	projReqs *prometheus.CounterVec
}

func (r rgetServer) handler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		domain, err := rgetwellknown.TrimDigestDomain(req.Host)
		if err != nil {
			fmt.Printf("request for unknown host %v unable to parse: %v\n", req.Host, err)
		}
		if len(domain) > 0 {
			r.projReqs.WithLabelValues(req.Method, domain).Inc()
		}
		http.Error(resp, "only POST is supported", http.StatusBadRequest)
		return
	}

	err := req.ParseForm()
	if err != nil {
		http.Error(resp, "invalid request", http.StatusBadRequest)
		return
	}

	sumsURL := req.Form.Get("url")
	fmt.Printf("submission: %v\n", sumsURL)

	// ensure the URL is coming from a host we know how to generate a
	// domain for by parsing it using the wellknown libraries
	domain, err := rgetwellknown.Domain(sumsURL)
	if err != nil {
		fmt.Printf("wellknown domain error: %v\n", err)
		resp.WriteHeader(http.StatusOK)
		return
	}

	r.projReqs.WithLabelValues(req.Method, domain).Inc()

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

	sums := rgethash.FromSHA256SumFile(string(sha256file))

	// Step 2: Save the file contents to the git repo by domain
	_, err = r.GitCache.Get(context.Background(), sums.Domain())
	if err == nil {
		// TODO(philips): add rate limiting and DDoS protections here
		fmt.Printf("cache hit: %v\n", sumsURL)
		resp.WriteHeader(http.StatusOK)
		return
	}

	// Step 3. Create the Certificate object for the domain and save that as well
	ctdomain := sums.Domain() + "." + domain
	err = r.GitCache.Put(context.Background(), ctdomain, sha256file)
	if err != nil {
		fmt.Printf("git put error: %v", err)
		http.Error(resp, "internal service error", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	return
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

	rs := rgetServer{
		GitCache: pubgc,
		projReqs: rr,
	}

	http.HandleFunc("/", rs.handler)

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
