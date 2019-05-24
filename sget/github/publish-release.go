package github

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

func AddCommands(root *cobra.Command) {
	root.AddCommand(publishReleaseCmd)
}

var publishReleaseCmd = &cobra.Command{
	Use:   "publish-release",
	Short: "Push a binary digests based on a previously created GitHub release",
	Long: `Use the GitHub API to download all assets related to a project
release and push digests into the binary transparency log:
`,
	Run: publishGithubMain,
}

func init() {

	publishReleaseCmd.Flags().StringP("owner", "o", "", "Repo owner name")
	viper.BindPFlag("owner", publishReleaseCmd.Flags().Lookup("owner"))

	publishReleaseCmd.Flags().StringP("repo", "r", "", "Repo name")
	viper.BindPFlag("repo", publishReleaseCmd.Flags().Lookup("repo"))

	publishReleaseCmd.Flags().StringP("tag", "t", "", "Release tag")
	viper.BindPFlag("tag", publishReleaseCmd.Flags().Lookup("tag"))

	publishReleaseCmd.Flags().BoolP("all-releases", "a", false, "Publish all releases")
	viper.BindPFlag("all-releases", publishReleaseCmd.Flags().Lookup("all-releases"))
}

func publishGithubMain(cmd *cobra.Command, args []string) {
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
