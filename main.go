package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/certificate-transparency-go/ctutil"
	"github.com/google/certificate-transparency-go/loglist"
	"github.com/google/certificate-transparency-go/x509"
	"github.com/google/certificate-transparency-go/x509util"
	"github.com/google/go-github/v24/github"
	"github.com/google/trillian/merkle"
	"github.com/google/trillian/merkle/rfc6962"
)

func main() {
	var (
		release string
		commit  string
	)

	if len(os.Args) < 2 {
		fmt.Printf("usage: sget file file file\n")
		os.Exit(1)
	}

	var chain []*x509.Certificate
	var valid, invalid int
	var domain string
	var totalInvalid int

	hc := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()
	lf := ctutil.NewLogInfo

	// TODO(philips): bump to ALlLogListURL
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

	domain = "https://google.com"

	// Get chain served online for TLS connection to site, and check any SCTs
	// provided alongside on the connection along the way.
	chain, valid, invalid, err = getAndCheckSiteChain(ctx, lf, domain, ll, hc)
	if err != nil {
		panic(fmt.Sprintf("%s: failed to get cert chain: %v", domain, err))
	}
	fmt.Printf("Found %d external SCTs for %q, of which %d were validated\n", (valid + invalid), domain, valid)
	totalInvalid += invalid

	// Check the chain for embedded SCTs.
	valid, invalid = checkChain(ctx, lf, chain, ll, hc)
	fmt.Printf("Found %d embedded SCTs for %q, of which %d were validated\n", (valid + invalid), domain, valid)
	totalInvalid += invalid

	if totalInvalid > 0 {
		panic("Invalid chain SCT found")
	}
}
