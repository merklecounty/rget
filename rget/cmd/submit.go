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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"go.merklecounty.com/rget/rgetgithub"
	"go.merklecounty.com/rget/rgetwellknown"
	"github.com/spf13/cobra"
)

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit [https://example.com/path/to/downloads/SHA256SUMS]",
	Short: "submit a URL to the recorder",
	Long: `The submitted URL will be fetched by the recorder service,
a record domain name will be generated, and a subsequent request to that
domain will cause a certificate to be generated and logged.`,
	Run: submit,
}

func init() {
	rootCmd.AddCommand(submitCmd)
}

func submit(cmd *cobra.Command, args []string) {
	resp, err := http.PostForm("https://"+rgetwellknown.PublicServiceHost+"/api/v1/submit", url.Values{
		"url": {args[0]},
	})
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("submit POST error: %v\n", err)
		os.Exit(1)
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("submit read response error: %v\n", err)
		os.Exit(1)
	}

	// TODO(philips): create a rgetwellknown function to generate a "test URL"
	m, err := rgetwellknown.GitHubMatches(args[0])
	if err != nil {
		fmt.Printf("unable to parse GitHub URL: %v\n", err)
		os.Exit(1)
	}

	aurls := rgetgithub.ArchiveURLs(m["org"], m["repo"], m["tag"])

	fmt.Printf("fetch a file for this submitted release by running:\n\n")
	fmt.Printf("rget %s\n", aurls[0])
}
