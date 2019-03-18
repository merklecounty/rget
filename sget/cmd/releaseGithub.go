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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v24/github"
	"github.com/google/trillian/merkle"
	"github.com/google/trillian/merkle/rfc6962"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context/ctxhttp"
)

// releaseGithubCmd represents the releaseGithub command
var releaseGithubCmd = &cobra.Command{
	Use:   "release-github",
	Short: "Push a binary digests based on a previously created GitHub release",
	Long: `Use the GitHub API to download all assets related to a project
release and push digests into the binary transparency log:
`,
	Run: releaseGithubMain,
}

func init() {
	rootCmd.AddCommand(releaseGithubCmd)

	releaseGithubCmd.Flags().StringP("owner", "o", "", "Repo owner name")
	viper.BindPFlag("owner", releaseGithubCmd.Flags().Lookup("owner"))

	releaseGithubCmd.Flags().StringP("repo", "r", "", "Repo name")
	viper.BindPFlag("repo", releaseGithubCmd.Flags().Lookup("repo"))

	releaseGithubCmd.Flags().StringP("tag", "t", "", "Release tag")
	viper.BindPFlag("tag", releaseGithubCmd.Flags().Lookup("tag"))

	releaseGithubCmd.Flags().BoolP("all-releases", "a", false, "Publish all releases")
	viper.BindPFlag("all-releases", releaseGithubCmd.Flags().Lookup("all-releases"))
}

func releaseGithubMain(cmd *cobra.Command, args []string) {
	var releases []github.RepositoryRelease

	client := github.NewClient(nil)
	ctx := context.Background()

	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	hc := &http.Client{Transport: tr}

	owner := viper.GetString("owner")
	repo := viper.GetString("repo")
	tag := viper.GetString("tag")

	if tag != "" {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
		if err != nil {
			panic(err)
		}
		releases = append(releases, *release)
	} else if viper.GetBool("all-releases") {
		var err error
		releases, err = allReleases(client, owner, repo)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Printf("error: no tag and --all-releases not set")
		os.Exit(1)
	}

	t := merkle.NewInMemoryMerkleTree(rfc6962.DefaultHasher)

	for _, r := range releases {
		urls := []string{}
		for _, a := range r.Assets {
			urls = append(urls, *a.BrowserDownloadURL)
		}
		urls = append(urls, *r.ZipballURL, *r.TarballURL)

		for _, u := range urls {
			resp, err := ctxhttp.Get(ctx, hc, u)
			if err != nil {
				panic(err)
			}
			sum, err := readerDigest(resp.Body)
			resp.Body.Close()
			if err != nil {
				panic(err)
			}

			fmt.Printf("%x  %s\n", sum, u)
			t.AddLeaf(sum)
		}
	}
	rh := t.CurrentRoot().Hash()
	fmt.Printf("merkle root: %s\n", hex.EncodeToString(rh))
	fmt.Printf("domain: %s\n", githubDomain(owner, repo, tag, rh))
}

func githubDomain(owner, repo, tag string, digest []byte) string {
	return fmt.Sprintf("%s.%s.%s.%s.%s.github.io.secured.dev", hex.EncodeToString(digest[:16]), hex.EncodeToString(digest[16:]), tag, repo, owner)
}

func allReleases(client *github.Client, owner string, repo string) ([]github.RepositoryRelease, error) {
	panic("TODO")
	opts := &github.ListOptions{Page: 0, PerPage: 1}
	rs, _, _ := client.Repositories.ListReleases(context.Background(), owner, repo, opts)

	var release string
	for _, r := range rs {
		release = *r.TagName
		println(release)
		for _, a := range r.Assets {
			println(*a.URL)
		}
	}
	return nil, nil
}

func readerDigest(r io.Reader) (sum []byte, err error) {
	h := sha256.New()

	_, err = io.Copy(h, r)
	if err != nil {
		panic(err)
	}
	sum = h.Sum(nil)

	return sum, nil
}
