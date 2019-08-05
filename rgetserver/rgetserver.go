package rgetserver

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/merklecounty/rget/gitcache"
	"github.com/merklecounty/rget/rgethash"
	"github.com/merklecounty/rget/rgetwellknown"
)

type Server struct {
	*gitcache.GitCache
	ProjReqs *prometheus.CounterVec
}

type release struct {
	Full  string
	Short string
}

var (
	releaseTemplate *template.Template
	rootTemplate    *template.Template
)

func init() {
	releaseTemplate = template.Must(template.New("release").Parse(`<html>
<head><title>{{.Short}} - Merkle County</title></head>
<body>
<h2>{{.Short}}</h2>
<ul>
  <li><a href="https://github.com/merklecounty/records/blob/master/{{.Full}}">Merkle County Record</a></li>
</ul>
</body>
</html>`))

	rootTemplate = template.Must(template.New("root").Parse(`<html>
<head><title>Merkle County</title></head>
<body>
<h2>Merkle County</h2>
<ul>
  <li><a href="https://merklecounty.substack.com">Newsletter and Blog</a></li>
  <li><a href="https://github.com/merklecounty/rget">GitHub</a></li>
  <li><a href="https://go.merklecounty.com">Go Packages</a></li>
</ul>
</body>
</html>`))

}

func (s Server) ReleaseHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(resp, "only GET is supported", http.StatusBadRequest)
		return
	}

	if req.Host == rgetwellknown.PublicServiceHost {
		rootTemplate.Execute(resp, nil)
		return
	}

	short, err := rgetwellknown.TrimDigestDomain(req.Host)
	if err != nil {
		fmt.Printf("request for unknown host %v unable to parse: %v\n", req.Host, err)
	}
	if len(short) > 0 {
		s.ProjReqs.WithLabelValues(req.Method, short).Inc()
	}

	full := strings.TrimSuffix(req.Host, "."+rgetwellknown.PublicServiceHost)

	r := &release{Full: full, Short: short}
	releaseTemplate.Execute(resp, r)

	return
}

func (r Server) APIHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(resp, "only POST is supported", http.StatusBadRequest)
		return
	}

	err := req.ParseForm()
	if err != nil {
		http.Error(resp, "invalid request", http.StatusBadRequest)
		return
	}

	sumsURL := req.Form.Get("url")
	fmt.Printf("submission: %v\n", sumsURL)

	// ensure the URL is coming from a host we know how to generate a
	// domain for by parsing it using the wellknown libraries
	domain, err := rgetwellknown.Domain(sumsURL)
	if err != nil {
		fmt.Printf("wellknown domain error: %v\n", err)
		resp.WriteHeader(http.StatusOK)
		return
	}

	r.ProjReqs.WithLabelValues(req.Method, domain).Inc()

	// Step 1: Download the SHA256SUMS that is correct for the URL
	response, err := http.Get(sumsURL)
	var sha256file []byte
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		var err error
		defer response.Body.Close()
		sha256file, err = ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
	}

	sums := rgethash.FromSHA256SumFile(string(sha256file))

	// Step 2: Save the file contents to the git repo by domain
	_, err = r.GitCache.Get(context.Background(), sums.Domain())
	if err == nil {
		// TODO(philips): add rate limiting and DDoS protections here
		fmt.Printf("cache hit: %v\n", sumsURL)
		resp.WriteHeader(http.StatusOK)
		return
	}

	// Step 3. Create the Certificate object for the domain and save that as well
	ctdomain := sums.Domain() + "." + domain
	err = r.GitCache.Put(context.Background(), ctdomain, sha256file)
	if err != nil {
		fmt.Printf("git put error: %v", err)
		http.Error(resp, "internal service error", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	return
}
