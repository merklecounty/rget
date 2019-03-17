package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/v24/github"
)

func main() {
	fmt.Println("vim-go")
	client := github.NewClient(nil)
	opts := &github.ListOptions{Page: 0, PerPage: 1}
	rs, _, _ := client.Repositories.ListReleases(context.Background(), "philips", "releases-test", opts)
	tags, _, _ := client.Repositories.ListTags(context.Background(), "philips", "releases-test", opts)

	for _, r := range rs {
		release := *r.TagName
		gt := *tags[0].GetCommit().SHA
		for _, a := range r.Assets {
			println(*a.URL)
		}
	}
}
