package sgethash

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

func (s URLSumList) GetURLSum(url string) *URLSum {
	for _, u := range s {
		if u.URL == url {
			return &u
		}
	}

	return nil
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
