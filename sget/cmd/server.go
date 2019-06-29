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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/philips/sget/autocert"
	"github.com/philips/sget/gitcache"
	"github.com/philips/sget/sgethash"
	"github.com/philips/sget/sgetwellknown"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "sget API server and TLS server",
	Long: `Provides an HTTP and HTTPS server that saves into two different
git repos one with TLS secrets and one with public data that can be audited.`,
	Run: server,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

type sumRepo gitcache.GitCache

func (r sumRepo) handler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
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

	sums := sgethash.FromSHA256SumFile(string(sha256file))

	// Step 2: Save the file contents to the git repo by domain
	gc := gitcache.GitCache(r)

	_, err = gc.Get(context.Background(), sums.Domain())
	if err == nil {
		// TODO(philips): add rate limiting and DDoS protections here
		fmt.Printf("cache hit: %v\n", sumsURL)
		resp.WriteHeader(http.StatusOK)
		return
	}
	err = gc.Put(context.Background(), sums.ShortDomain(), sha256file)
	if err != nil {
		fmt.Printf("git put error: %v\n", err)
		http.Error(resp, "internal service error", http.StatusInternalServerError)
		return
	}

	err = gc.Put(context.Background(), sums.Domain(), sha256file)
	if err != nil {
		fmt.Printf("git put error: %v\n", err)
		http.Error(resp, "internal service error", http.StatusInternalServerError)
		return
	}

	// Step 3. Create the Certificate object for the domain and save that as well
	domain, err := sgetwellknown.Domain(sumsURL)
	if err != nil {
		fmt.Printf("wellknown domain error: %v\n", err)
		resp.WriteHeader(http.StatusOK)
		return
	}
	ctdomain := sums.Domain() + "." + domain
	err = gc.Put(context.Background(), ctdomain, sha256file)
	if err != nil {
		fmt.Printf("git put error: %v", err)
		http.Error(resp, "internal service error", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	return
}

const tld = ".established.ifup.org"

func server(cmd *cobra.Command, args []string) {
	pubgit := args[0]
	privgit := args[1]

	pubgc, err := gitcache.NewGitCache(pubgit, "public")
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", sumRepo(*pubgc).handler)

	privgc, err := gitcache.NewGitCache(privgit, "private")
	if err != nil {
		panic(err)
	}

	hostPolicyNoLog := func(ctx context.Context, host string) ([]string, error) {
		if tld == host {
			return nil, nil
		}

		if !strings.HasSuffix(host, tld) {
			return nil, errors.New(fmt.Sprintf("not in TLD %v", tld))
		}

		key := strings.TrimSuffix(host, tld)

		if strings.Contains(key, ".") {
			return nil, errors.New(fmt.Sprintf("common name cannot have subdomains %v", tld))
		}

		_, err := pubgc.Get(ctx, key)
		if err != nil {
			fmt.Printf("unknown merkle prefix %v for %v\n", key, host)
			// TODO(philips): leak a nicer error
			return nil, err
		}

		matches, err := pubgc.Prefix(ctx, key)
		if err != nil {
			return nil, err
		}

		fmt.Printf("%v SANs %v\n", key, matches)

		return matches, nil
	}

	hostPolicy := func(ctx context.Context, host string) ([]string, error) {
		fmt.Printf("hostPolicy called %v\n", host)
		sans, err := hostPolicyNoLog(ctx, host)
		fmt.Printf("hostPolicy err %v\n", err)
		return sans, err
	}

	m := &autocert.Manager{
		Cache:      privgc,
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Email:      "brandon@ifup.org",
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

	log.Fatal(http.ListenAndServe(":http", nil))
}
