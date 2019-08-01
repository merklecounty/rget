package rgethash

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/merklecounty/rget/autocert"
	"github.com/merklecounty/rget/gitcache"
	"github.com/merklecounty/rget/internal/testutil"
	"github.com/merklecounty/rget/rgetwellknown"
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

func TestHostPolicy(t *testing.T) {
	testCases := []struct {
		put     bool
		input   string
		value   []byte
		expect  autocert.Policy
		wantErr error
	}{
		{
			true,
			"cd83cf8f413393d30d53626c5f17f9ae.af90c0ab70bffa127b54cdb82d3c1499.v0-0-6.rget.merklecounty.github.com",
			[]byte(`d4cb7fc206cbd147b3397c1e1b88513831c9780fc9675bebc300112365979465  rget-v0.0.6-linux-arm.tar.gz
7239591ab580c911738130cc62d8a5cd9c6c05c79fa7abfdf32bad0d68a70844  rget-v0.0.6-windows-amd64.tar.gz
18908181de67376c12b7e34de7c3e4aeaddc24cebab8c7d8115cf31dfbe236f2  rget-v0.0.6-linux-amd64.tar.gz
38c6ee23c7f5fbdc7ef207dda25d8e030c8300fa94d71b6b4adc878af9343ba8  rget-v0.0.6-linux-arm64.tar.gz
5b64ee638b847ca72dc1d029437d69e987d439ff420135241687975c6ca2484a  rget-v0.0.6-darwin-amd64.tar.gz`),
			autocert.Policy{
				CommonName: "cd83cf8f413393d30d53626c5f17f9ae.recorder.merklecounty.com",
				DNSNames: []string{
					"cd83cf8f413393d30d53626c5f17f9ae.af90c0ab70bffa127b54cdb82d3c1499.v0-0-6.rget.merklecounty.github.com.recorder.merklecounty.com",
					"cd83cf8f413393d30d53626c5f17f9ae.af90c0ab70bffa127b54cdb82d3c1499.recorder.merklecounty.com",
				},
			},
			nil,
		},
		{
			false,
			"67568fff9faa4928c8bd4dd4aeb1a31d.74e766a21475966740ecbf12685e6821cd83cf8f413393d30d53626c5f17f9ae.af90c0ab70bffa127b54cdb82d3c1499.v0-0-6.rget.merklecounty.github.com",
			[]byte{1},
			autocert.Policy{},
			ErrUnknownMerklePrefix,
		},
	}

	dir, err := ioutil.TempDir("", "TestHostPolicy")
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(dir)

	url := filepath.Join(dir, "repo")

	gitURL := testutil.EmptyGitRepo(t, url)

	gc, err := gitcache.NewGitCache(gitURL, nil, filepath.Join(dir, "cache"))
	if err != nil {
		t.Fatal(err)
	}

	hp := HostPolicyFunc(gc)

	ctx := context.Background()

	for ti, tt := range testCases {
		if tt.put {
			if err := gc.Put(ctx, tt.input, tt.value); err != nil {
				t.Errorf("%d: put domain %v: %v", ti, tt.input, err)
			}
		}

		p, err := hp(ctx, tt.input+"."+rgetwellknown.PublicServiceHost)
		switch {
		case err != tt.wantErr:
			t.Fatalf("%d: want %v got %v %v", ti, tt.wantErr, err, p)
		case tt.wantErr != nil:
			continue
		case err != nil:
			t.Errorf("%d: policy %v: %v", ti, tt.input, err)
		}

		if !reflect.DeepEqual(tt.expect, p) {
			t.Errorf("%d: policies don't match want\n\t%v\n got\n\t%v\n", ti, tt.expect, p)
		}
	}
}
