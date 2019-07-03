package rgetwellknown

import (
	"testing"
)

func TestDomain(t *testing.T) {
	testCases := []struct {
		downloadURL string
		wantDomain  string
		wantErr     bool
	}{
		{"https://api.github.com/repos/philips/releases-test/zipball/v2.0", "v2-0.releases-test.philips.github.com", false},
		{"https://github.com/philips/releases-test/archive/v2.0.zip", "v2-0.releases-test.philips.github.com", false},
		{"https://github.com/philips/releases-test/archive/v2.0.tar.gz", "v2-0.releases-test.philips.github.com", false},
		{"https://github.com/philips/releases-test/releases/download/v2.0/SHA256SUMS", "v2-0.releases-test.philips.github.com", false},
	}

	for ti, tt := range testCases {
		dd, err := Domain(tt.downloadURL)
		if err != nil {
			t.Errorf("%d: error from downloadURL %v: %v", ti, tt.downloadURL, err)
		}

		if dd != tt.wantDomain {
			t.Errorf("%d: domain %v != %v", ti, dd, tt.wantDomain)
		}
	}
}

func TestSumPrefix(t *testing.T) {
	testCases := []struct {
		downloadURL string
		wantDomain  string
		wantErr     bool
	}{
		{"https://api.github.com/repos/philips/releases-test/zipball/v2.0", "https://github.com/philips/releases-test/releases/download/v2.0/", false},
		{"https://github.com/philips/releases-test/archive/v2.0.zip", "https://github.com/philips/releases-test/releases/download/v2.0/", false},
		{"https://github.com/philips/releases-test/releases/download/v2.0/SHA256SUMS", "https://github.com/philips/releases-test/releases/download/v2.0/", false},
	}

	for ti, tt := range testCases {
		dd, err := SumPrefix(tt.downloadURL)
		if err != nil {
			t.Errorf("%d: error from downloadURL %v: %v", ti, tt.downloadURL, err)
		}

		if dd != tt.wantDomain {
			t.Errorf("%d: domain %v != %v", ti, dd, tt.wantDomain)
		}
	}
}
