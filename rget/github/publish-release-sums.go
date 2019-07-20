package github

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/google/go-github/v24/github"
	"github.com/nmrshll/oauth2-noserver"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/merklecounty/rget/rgetgithub"
	"github.com/merklecounty/rget/rgethash"
	"github.com/merklecounty/rget/rgetwellknown"
)

var publishReleaseSumsCmd = &cobra.Command{
	Use:   "publish-release-sums [github releases URL]",
	Short: "Publish the release sums file for a release to a SHA256SUMS file",
	Long: `
`,
	Run: publishReleaseSumsMain,
}

func init() {
	publishReleaseSumsCmd.Flags().BoolP("dry-run", "d", false, "Do not upload file to GitHub")
}

func publishReleaseSumsMain(cmd *cobra.Command, args []string) {
	var releases []github.RepositoryRelease

	ctx := context.Background()

	if len(args) != 1 {
		cmd.Usage()
		os.Exit(1)
	}

	m, err := rgetwellknown.GitHubMatches(args[0])
	if err != nil {
		fmt.Printf("matches: %v\n", err)
		os.Exit(1)
	}

	conf := &oauth2.Config{
		ClientID:     "81b93ee2e0d70958d933",
		ClientSecret: "86e236464cdb40b07b085ade131b41b156f29c62",
		Scopes:       []string{"repo"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		panic(err)
	}

	client := github.NewClient(nil)

	if !dryRun {
		tc, err := oauth2ns.AuthenticateUser(conf)
		if err != nil {
			log.Fatal(err)
		}
		client = github.NewClient(tc.Client)
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

	urls := rgethash.URLSumList{}
	for _, r := range releases {
		for _, a := range r.Assets {
			urls.AddURL(*a.BrowserDownloadURL)
		}
		urls.AddURL(*r.ZipballURL)
		urls.AddURL(*r.TarballURL)

		// Grab the git tag URLs
		tu := rgetgithub.ArchiveURLs(m["org"], m["repo"], *r.TagName)
		for _, t := range tu {
			urls.AddURL(t)
		}

		sha256sumfile := urls.SHA256SumFile()

		content := []byte(sha256sumfile)

		fmt.Printf("generated SHA256SUMS:\n\n%s\n", content)

		if !dryRun {
			uploadSums(client, m["org"], m["repo"], *r.TagName, r, content)
			fmt.Printf("submit the uploaded SHA256SUMS to the public record by running:\n\nrget submit https://github.com/%s/%s/releases/download/%s/SHA256SUMS\n", m["org"], m["repo"], *r.TagName)
		}
	}

}

func uploadSums(client *github.Client, owner, repo, tag string, release github.RepositoryRelease, content []byte) {
	ctx := context.Background()

	tmpfile, err := ioutil.TempFile("", "rget*")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	// github client, rightfully, expects to be at the beginning of the file
	tmpfile.Sync()
	tmpfile.Seek(0, 0)

	uo := github.UploadOptions{Name: "SHA256SUMS"}
	_, _, err = client.Repositories.UploadReleaseAsset(ctx, owner, repo, *release.ID, &uo, tmpfile)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
