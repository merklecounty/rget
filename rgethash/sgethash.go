package rgethash

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/trillian/merkle"
	"github.com/google/trillian/merkle/rfc6962"
	"golang.org/x/net/context/ctxhttp"

	"github.com/merklecounty/rget/autocert"
	"github.com/merklecounty/rget/gitcache"
	"github.com/merklecounty/rget/rgetwellknown"
)

type URLSum struct {
	URL string
	Sum []byte
}

type URLSumList []URLSum

func FromSHA256SumFile(file string) URLSumList {
	s := bufio.NewScanner(strings.NewReader(file))

	list := URLSumList{}
	for s.Scan() {
		u := URLSum{}
		fmt.Sscanf(s.Text(), "%x  %s", &u.Sum, &u.URL)
		list = append(list, u)
	}

	return list
}

func (s *URLSumList) AddURL(url string) error {
	ctx := context.Background()
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	hc := &http.Client{Transport: tr}

	resp, err := ctxhttp.Get(ctx, hc, url)
	if err != nil {
		return err
	}
	sum, err := readerDigest(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	*s = append(*s, URLSum{url, sum})

	return nil
}

func (s URLSumList) Domain() string {
	root := s.MerkleRoot()
	return fmt.Sprintf("%s.%s", hex.EncodeToString(root[:16]), hex.EncodeToString(root[16:]))
}

func (s URLSumList) ShortDomain() string {
	root := s.MerkleRoot()
	return fmt.Sprintf("%s", hex.EncodeToString(root[:16]))
}

func (s URLSumList) GetURLSum(url string) *URLSum {
	for _, u := range s {
		if u.URL == url {
			return &u
		}
	}

	return nil
}

func (s URLSumList) SumExists(sum []byte) bool {
	for _, u := range s {
		if bytes.Compare(sum, u.Sum) == 0 {
			return true
		}
	}

	return false
}

func (s URLSumList) MerkleRoot() []byte {
	t := merkle.NewInMemoryMerkleTree(rfc6962.DefaultHasher)

	for _, u := range s {
		t.AddLeaf(u.Sum)
	}
	return t.CurrentRoot().Hash()
}

func (s URLSumList) SHA256SumFile() string {
	var buf bytes.Buffer
	for _, u := range s {
		buf.Write([]byte(fmt.Sprintf("%x  %s\n", u.Sum, u.URL)))
	}
	return buf.String()

}

func readerDigest(r io.Reader) (sum []byte, err error) {
	h := sha256.New()

	_, err = io.Copy(h, r)
	if err != nil {
		panic(err)
	}
	sum = h.Sum(nil)

	return sum, nil
}

var (
	ErrCommonNameEmpty     error
	ErrUnknownMerklePrefix error
)

func init() {
	ErrCommonNameEmpty = fmt.Errorf("common name empty")
	ErrUnknownMerklePrefix = fmt.Errorf("unknown merkle prefix")
}

// HostPolicyFunc returns a HostPolicy that returns Policies
// based on Sums that exist in the GitCache repo
func HostPolicyFunc(pubgc *gitcache.GitCache) autocert.HostPolicy {
	hostPolicy := func(ctx context.Context, host string) (autocert.Policy, error) {
		if rgetwellknown.PublicServiceHost == host {
			return autocert.Policy{CommonName: host}, nil
		}

		if !strings.HasSuffix(host, "."+rgetwellknown.PublicServiceHost) {
			return autocert.Policy{}, fmt.Errorf("not in TLD %v", rgetwellknown.PublicServiceHost)
		}

		key := strings.TrimSuffix(host, "."+rgetwellknown.PublicServiceHost)

		// Reduce to the shortest domain
		parts := strings.Split(key, ".")
		if len(parts) == 0 {
			return autocert.Policy{}, ErrCommonNameEmpty
		}
		key = parts[0]

		matches, err := pubgc.Prefix(ctx, key)
		if err != nil {
			return autocert.Policy{}, err
		}

		if len(matches) != 1 {
			return autocert.Policy{}, ErrUnknownMerklePrefix
		}

		content, err := pubgc.Get(ctx, matches[0])
		if err != nil {
			fmt.Printf("unknown merkle prefix %v for %v\n", key, host)
			// TODO(philips): leak a nicer error
			return autocert.Policy{}, err
		}

		sums := FromSHA256SumFile(string(content))

		p := autocert.Policy{
			CommonName: sums.ShortDomain() + "." + rgetwellknown.PublicServiceHost,
			DNSNames: []string{
				matches[0] + "." + rgetwellknown.PublicServiceHost,
				sums.Domain() + "." + rgetwellknown.PublicServiceHost,
			},
		}

		return p, nil
	}

	return hostPolicy
}
