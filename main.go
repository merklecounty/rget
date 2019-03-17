package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

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

func decodeHexStringOrPanic(hs string) []byte {
	data, err := hex.DecodeString(hs)
	if err != nil {
		panic(fmt.Errorf("failed to decode test data: %s", hs))
	}

	return data
}
