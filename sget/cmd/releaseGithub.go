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
	"os"
	"strings"

	"github.com/google/go-github/v24/github"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/philips/sget/sgethash"
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

	urls := sgethash.URLSumList{}
	for _, r := range releases {
		for _, a := range r.Assets {
			urls.AddURL(*a.BrowserDownloadURL)
		}
		urls.AddURL(*r.ZipballURL)
		urls.AddURL(*r.TarballURL)
	}

	sha256sumfile := urls.SHA256SumFile()
	fmt.Printf("%v\n", sha256sumfile)

	sh := shell.NewShell("localhost:5001")
	cid, err := sh.Add(strings.NewReader(sha256sumfile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("added %s", cid)
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
