package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v24/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/philips/sget/sgethash"
)

var generateReleaseSumsCmd = &cobra.Command{
	Use:   "generate-release-sums",
	Short: "Generate the release sums file for a release",
	Long: `
`,
	Run: generateReleaseSumsMain,
}

func generateReleaseSumsMain(cmd *cobra.Command, args []string) {
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
}
