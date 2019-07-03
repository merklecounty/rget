package rgethash

import (
	"testing"
)

func TestDomain(t *testing.T) {
	testCases := []struct {
		ul      URLSumList
		domain  string
		wantErr bool
	}{
		{[]URLSum{{URL: "https://example.com", Sum: []byte("f0a15079480da8c6ba74ac40dbb51443b009abc210a2a363e4dba7173b2adab5")}}, "30a44a43b5d07d2a1574668ea941afee.70b20c0eb9a1b31d16486519ebb110ed", false},
	}

	for ti, tt := range testCases {
		if tt.ul.Domain() != tt.domain {
			t.Errorf("%d: domain %v != %v", ti, tt.ul.Domain(), tt.domain)
		}

		f := tt.ul.SHA256SumFile()
		nul := FromSHA256SumFile(f)
		t.Log(f)
		t.Log(nul.SHA256SumFile())

		if tt.ul.Domain() != nul.Domain() {
			t.Errorf("%d: sha256sum domain %v != %v", ti, tt.ul.Domain(), nul.Domain())
		}
	}
}
