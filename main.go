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

	client := github.NewClient(nil)
	opts := &github.ListOptions{Page: 0, PerPage: 1}
	rs, _, _ := client.Repositories.ListReleases(context.Background(), "philips", "releases-test", opts)
	tags, _, _ := client.Repositories.ListTags(context.Background(), "philips", "releases-test", opts)

	for _, r := range rs {
		release = *r.TagName
		commit = *tags[0].GetCommit().SHA
		println(release)
		println(commit)
		for _, a := range r.Assets {
			println(*a.URL)
		}
	}

	t := merkle.NewInMemoryMerkleTree(rfc6962.DefaultHasher)
	println(hex.EncodeToString(t.CurrentRoot().Hash()))
	//t.AddLeaf([]byte(decodeHexStringOrPanic("")))
	//t.AddLeaf([]byte(decodeHexStringOrPanic("00")))
	//t.AddLeaf([]byte(decodeHexStringOrPanic("10")))
	//println(hex.EncodeToString(t.CurrentRoot().Hash()))

	var sha256sums string
	for _, name := range os.Args[1:] {
		_, sum := hashFile(name)
		sha256sums = sha256sums + fmt.Sprintf("%x  %s\n", sum, name)
		t.AddLeaf(sum)
	}

	fmt.Println(sha256sums)

	rh := t.CurrentRoot().Hash()
	fmt.Printf("merkle root: %s\n", hex.EncodeToString(rh))
	fmt.Printf("domain: %s.%s.%s.sget.philips.github.io.secured.dev", hex.EncodeToString(rh[:16]), hex.EncodeToString(rh[16:]), release)

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

func hashFile(name string) (err error, sum []byte) {
	h := sha256.New()
	file, err := os.Open(name)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", name, err))
	}

	_, err = io.Copy(h, file)
	if err != nil {
		panic(err)
	}
	sum = h.Sum(nil)

	return nil, sum
}
