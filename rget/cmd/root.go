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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cavaliercoder/grab"
	"github.com/google/certificate-transparency-go/ctutil"
	"github.com/google/certificate-transparency-go/loglist"
	"github.com/google/certificate-transparency-go/x509"
	"github.com/google/certificate-transparency-go/x509util"

	"github.com/merklecounty/rget/rgetct"
	"github.com/merklecounty/rget/rgethash"
	"github.com/merklecounty/rget/rgetwellknown"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rget [URL]",
	Short: "Get a URL and verify the contents with CT Log backed transparency",
	Long: `rget is similar to other popular URL fetchers with an additional layer of security.
By using the Certificate Transparency Log infrastructure that enables third-party auditing of
the web's certificate authority infrastructure rget can give you strong guarantees that the
cryptographic hash digest of the binary you are downloading appears in a public log.
`,

	Args: cobra.ExactArgs(1),

	Run: get,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rget.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".rget" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".rget")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func get(cmd *cobra.Command, args []string) {
	var chain []*x509.Certificate
	var valid, invalid int
	var totalInvalid int

	durl := args[0]

	// Step 1: Download the SHA256SUMS that is correct for the URL
	prefix, err := rgetwellknown.SumPrefix(durl)
	sumsURL := prefix + "SHA256SUMS"
	fmt.Printf("Downloading sums file: %v\n", sumsURL)
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

	// Step 2. Generate the CT URL from the SHA256SUMS file
	domain, err := rgetwellknown.Domain(durl)
	if err != nil {
		fmt.Printf("wellknown domain error: %v", err)
		os.Exit(1)
	}

	sums := rgethash.FromSHA256SumFile(string(sha256file))
	cturl := "https://" + sums.Domain() + "." + domain + "." + rgetwellknown.PublicServiceHost

	hc := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()
	lf := ctutil.NewLogInfo

	// TODO(philips): bump to AllLogListURL and embed into this code instead of relying on Google
	llData, err := x509util.ReadFileOrURL(loglist.LogListURL, hc)
	if err != nil {
		fmt.Printf("Failed to read log list: %v", err)
		os.Exit(1)
	}
	ll, err := loglist.NewFromJSON(llData)
	if err != nil {
		fmt.Printf("Failed to parse log list: %v", err)
		os.Exit(1)
	}

	// Get chain served online for TLS connection to site, and check any SCTs
	// provided alongside on the connection along the way.
	chain, valid, invalid, err = rgetct.GetAndCheckSiteChain(ctx, lf, cturl, ll, hc)
	if err != nil {
		fmt.Printf("%s: failed to get cert chain: %v\n", cturl, err)
		os.Exit(1)
	}
	fmt.Printf("Found %d external SCTs for %q, of which %d were validated\n", (valid + invalid), cturl, valid)
	totalInvalid += invalid

	// Check the chain for embedded SCTs.
	valid, invalid = rgetct.CheckChain(ctx, lf, chain, ll, hc)
	fmt.Printf("Found %d embedded SCTs for %q, of which %d were validated\n", (valid + invalid), domain, valid)
	totalInvalid += invalid

	// create download request
	req, err := grab.NewRequest("", durl)
	if err != nil {
		fmt.Printf("failed to create grab request: %v\n", err)
		os.Exit(1)
	}
	req.NoCreateDirectories = true

	req.AfterCopy = func(resp *grab.Response) (err error) {
		var f *os.File
		f, err = os.Open(resp.Filename)
		if err != nil {
			return
		}
		defer func() {
			f.Close()
		}()

		h := sha256.New()
		_, err = io.Copy(h, f)
		if err != nil {
			return err
		}

		fileSum := h.Sum(nil)

		if !sums.SumExists(fileSum) {
			mmErr := fmt.Errorf("cannot find %x in %v list", fileSum, cturl)

			if err := os.Remove(resp.Filename); err != nil {
				// err should be os.PathError and include file path
				return fmt.Errorf(
					"cannot remove downloaded file with checksum mismatch: %v",
					mmErr)
			}

			return mmErr
		}

		req.SetChecksum(sha256.New(), fileSum, true)

		return
	}

	// download and validate file
	resp := grab.DefaultClient.Do(req)
	if err := resp.Err(); err != nil {
		fmt.Printf("Failed to grab: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Download validated and saved to", resp.Filename)
}
