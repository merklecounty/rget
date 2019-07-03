package gitcache

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/merklecounty/rget/autocert"
)

func TestDirCache(t *testing.T) {
	dir, err := ioutil.TempDir("", "autocert")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dir = filepath.Join(dir, "certs") // a nonexistent dir
	cache := autocert.DirCache(dir)
	ctx := context.Background()

	// test prefix
	expected := []string{"dummy1", "dummy1.dummy1", "dummy1.dummy1.dummy1"}
	for _, n := range expected {
		if err := cache.Put(ctx, n, []byte{1}); err != nil {
			t.Fatalf("put: %v", err)
		}
	}

	matches, err := prefix(cache, "dummy")
	if err != nil {
		t.Fatalf("prefix: %v", err)
	}

	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("matches = %v; want %v", matches, expected)
	}

}
