package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v24/github"
	"github.com/spf13/cobra"

	"github.com/philips/sget/sgethash"
	"github.com/philips/sget/sgetwellknown"
)

var generateReleaseSumsCmd = &cobra.Command{
	Use:   "generate-release-sums [github release URL]",
	Short: "Generate the release sums file for a GitHub release",
	Long: `Download all of the binaries for a release and generate a SHA256SUMS file
which is printed out to stdout.

example:
  sget generate-release-sums https://github.com/github/hub/releases/tag/v2.12.1
`,
	Run: generateReleaseSumsMain,
}

func generateReleaseSumsMain(cmd *cobra.Command, args []string) {
	var releases []github.RepositoryRelease

	client := github.NewClient(nil)
	ctx := context.Background()

	if len(args) != 1 {
		cmd.Usage()
		os.Exit(1)
	}

	m, err := sgetwellknown.GitHubMatches(args[0])
	if err != nil {
		fmt.Printf("matches: %v\n", err)
		os.Exit(1)
	}

	if m["tag"] != "" {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, m["org"], m["repo"], m["tag"])
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
