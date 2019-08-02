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
		{"https://github.com/philips/releases-test/archive/v2.0+nosums.zip", "v2-0-nosums.releases-test.philips.github.com", false},
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

func TestTrimDigestDomain(t *testing.T) {
	testCases := []struct {
		domain  string
		want    string
		wantErr bool
	}{
		{"2fcd82bbae7bcf7c0b0c5a2f91d3dd93.1e7c7be8587808ee85b347412ffa7514.v0-0-4.rget.merklecounty.github.com.recorder.merklecounty.com", "v0-0-4.rget.merklecounty.github.com", false},
		// digest too short
		{"1.2.v0-0-4.rget.merklecounty.github.com.recorder.merklecounty.com", "", true},
		// domain too short
		{"2fcd82bbae7bcf7c0b0c5a2f91d3dd93.1e7c7be8587808ee85b347412ffa7514.recorder.merklecounty.com", "", true},
		// wrong domain
		{"2fcd82bbae7bcf7c0b0c5a2f91d3dd93.1e7c7be8587808ee85b347412ffa7514.example.com", "", true},
	}

	for ti, tt := range testCases {
		dd, err := TrimDigestDomain(tt.domain)
		if !tt.wantErr && err != nil {
			t.Errorf("%d: error from TrimDigestDomain %v: %v", ti, tt.domain, err)
		}

		if tt.wantErr {
			if err == nil {
				t.Errorf("%d: wanted err got nil", ti)
			}
			continue
		}

		if dd != tt.want {
			t.Errorf("%d: domain %v != %v", ti, dd, tt.want)
		}
	}

}
