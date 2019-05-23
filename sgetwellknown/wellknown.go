package sgetwellknown

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// A vcsPath describes how to convert an import path into a
// version control system and repository name.
type vcsPath struct {
	prefix string         // prefix this description applies to
	regexp *regexp.Regexp // compiled pattern for import path
	domain string         // domain that will be used for the URL for the CT Log
}

// vcsPaths defines the meaning of import paths referring to
// commonly-used VCS hosting sites (github.com/user/dir)
// and import paths referring to a fully-qualified importPath
// containing a VCS type (foo.com/repo.git/dir)
var vcsPaths = []*vcsPath{
	// Github API
	{
		prefix: "api.github.com/",
		// https://api.github.com/repos/philips/releases-test/zipball/v2.0
		regexp: regexp.MustCompile(`^api\.(?P<root>github\.com)/repos/(?P<org>[A-Za-z0-9_.\-]+)/(?P<repo>[A-Za-z0-9_.\-]+)/(zipball|tarball)/(?P<tag>[A-Za-z0-9_.\-]+)$`),
		domain: "{tag}.{repo}.{org}.{root}",
	},
	// Github
	{
		prefix: "github.com/",
		// https://github.com/philips/releases-test/archive/v2.0.zip
		regexp: regexp.MustCompile(`^(?P<root>github\.com)/(?P<org>[A-Za-z0-9_.\-]+)/(?P<repo>[A-Za-z0-9_.\-]+)/archive/(?P<tag>[A-Za-z0-9_.\-]+)\.(zip|tar\.gz)$`),
		domain: "{tag}.{repo}.{org}.{root}",
	},
}

// Domain takes a target URL and returns the CT domain that should contain
// the SHA256Sum for this target.
// TODO(philips): define a well-known URL format to do this
// TODO(philips): handle docker URLs
func Domain(target string) (string, error) {
	return domainFromURL(target, vcsPaths)
}

var errUnknownSite = errors.New("no domain translation logic for this URL")

// expand rewrites s to replace {k} with match[k] for each key k in match.
func expand(match map[string]string, s string) string {
	// We want to replace each match exactly once, and the result of expansion
	// must not depend on the iteration order through the map.
	// A strings.Replacer has exactly the properties we're looking for.
	oldNew := make([]string, 0, 2*len(match))
	for k, v := range match {
		oldNew = append(oldNew, "{"+k+"}", v)
	}
	return strings.NewReplacer(oldNew...).Replace(s)
}

// domainFromURL takes a target download URL and builds a domain scheme that can be
// prepended with a merkle root to resolve to a certificate
func domainFromURL(downloadURL string, vcsPaths []*vcsPath) (string, error) {
	downloadPath := strings.TrimPrefix(downloadURL, "https://")

	for _, srv := range vcsPaths {
		if !strings.HasPrefix(downloadPath, srv.prefix) {
			continue
		}
		m := srv.regexp.FindStringSubmatch(downloadPath)
		if m == nil {
			if srv.prefix != "" {
				return "", fmt.Errorf("invalid %s domain path %q", srv.prefix, downloadPath)
			}
			continue
		}

		// Build map of named subexpression matches for expand.
		match := map[string]string{
			"prefix": srv.prefix,
		}
		for i, name := range srv.regexp.SubexpNames() {
			if name != "" && match[name] == "" {
				match[name] = m[i]
			}
		}
		if srv.domain != "" {
			match["domain"] = expand(match, srv.domain)
		}
		return match["domain"], nil
	}
	return "", errUnknownSite
}
