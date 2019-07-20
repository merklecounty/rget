package rgetgithub

import (
	"fmt"
	"net/url"
)

// ArchiveURLs generates source archive URLs for a GitHub repo tag
// e.g. https://github.com/philips/releases-test/archive/v1.0.zip and
// https://github.com/philips/releases-test/archive/v1.0.tar.gz
func ArchiveURLs(owner, repo, tag string) (urls []string) {
	u := url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   fmt.Sprintf("/%s/%s/archive/%s", owner, repo, tag),
	}
	urls = append(urls, u.String()+".tar.gz")
	urls = append(urls, u.String()+".zip")

	return
}
