package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/google/go-github/v24/github"
	"github.com/google/trillian/merkle"
	"github.com/google/trillian/merkle/rfc6962"
)

func main() {
	fmt.Println("vim-go")
	client := github.NewClient(nil)
	opts := &github.ListOptions{Page: 0, PerPage: 1}
	rs, _, _ := client.Repositories.ListReleases(context.Background(), "philips", "releases-test", opts)
	tags, _, _ := client.Repositories.ListTags(context.Background(), "philips", "releases-test", opts)

	for _, r := range rs {
		release := *r.TagName
		println(release)
		gt := *tags[0].GetCommit().SHA
		println(gt)
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
}

func decodeHexStringOrPanic(hs string) []byte {
	data, err := hex.DecodeString(hs)
	if err != nil {
		panic(fmt.Errorf("failed to decode test data: %s", hs))
	}

	return data
}
