package rgetgithub

import "testing"

func TestArchiveURLs(t *testing.T) {
	testCases := []struct {
		urls []string
		org  string
		repo string
		tag  string
	}{
		{
			urls: []string{
				"https://github.com/philips/releases-test/archive/v1.0.zip",
				"https://github.com/philips/releases-test/archive/v1.0.tar.gz",
			},
			org:  "philips",
			repo: "releases-test",
			tag:  "v1.0",
		},
	}

	for ti, tt := range testCases {
		aurls := ArchiveURLs(tt.org, tt.repo, tt.tag)

		for _, u := range tt.urls {
			if !contains(aurls, u) {
				t.Errorf("%d: %v not in %v", ti, u, aurls)
			}
		}
	}

}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
