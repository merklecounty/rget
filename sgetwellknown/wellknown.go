package sgetwellknown

import (
	"errors"
	"path"
	"regexp"
	"strings"
)

// A vcsPath describes how to convert an import path into a
// version control system and repository name.
type vcsPath struct {
	prefix    string         // prefix this description applies to
	regexp    *regexp.Regexp // compiled pattern for import path
	domain    string         // domain that will be used for the URL for the CT Log
	sumPrefix string         // URL prefix for a SUMS file
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
		regexp:    regexp.MustCompile(`^api\.(?P<root>github\.com)/repos/(?P<org>[A-Za-z0-9_.\-]+)/(?P<repo>[A-Za-z0-9_.\-]+)/(zipball|tarball)/(?P<tag>[A-Za-z0-9_.\-]+)$`),
		domain:    "{dnstag}.{repo}.{org}.{root}",
		sumPrefix: "https://github.com/{org}/{repo}/releases/download/{tag}/",
	},
	// Github release downloads
	{
		prefix: "github.com/",
		// https://github.com/philips/releases-test/releases/download/v2.0/SHA256SUMS
		regexp:    regexp.MustCompile(`^(?P<root>github\.com)/(?P<org>[A-Za-z0-9_.\-]+)/(?P<repo>[A-Za-z0-9_.\-]+)/releases/download/(?P<tag>[A-Za-z0-9_.\-]+)/(?P<file>[A-Za-z0-9_.\-]+)$`),
		domain:    "{dnstag}.{repo}.{org}.{root}",
		sumPrefix: "https://github.com/{org}/{repo}/releases/download/{tag}/",
	},

	// Github automatic archives
	{
		prefix: "github.com/",
		// https://github.com/philips/releases-test/archive/v2.0.zip
		regexp:    regexp.MustCompile(`^(?P<root>github\.com)/(?P<org>[A-Za-z0-9_.\-]+)/(?P<repo>[A-Za-z0-9_.\-]+)/archive/(?P<tag>[A-Za-z0-9_.\-]+)\.(zip|tar\.gz)$`),
		domain:    "{dnstag}.{repo}.{org}.{root}",
		sumPrefix: "https://github.com/{org}/{repo}/releases/download/{tag}/",
	},
}

// Domain takes a target URL and returns the domain postfix to be appended to a
// URLSumList.Domain()
// TODO(philips): define a well-known URL format to do this
// TODO(philips): handle docker URLs
func Domain(target string) (string, error) {
	match, err := matchesFromURL(target, vcsPaths)
	if err != nil {
		return "", err
	}
	return match["domain"], nil
}

// SumPrefix takes a target URL and returns the URL prefix for
// the SHA256SUMS file for the target object.
func SumPrefix(target string) (string, error) {
	match, err := matchesFromURL(target, vcsPaths)
	if err != nil {
		return "", err
	}
	return match["sumPrefix"], nil
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
func matchesFromURL(downloadURL string, vcsPaths []*vcsPath) (map[string]string, error) {
	downloadPath := strings.TrimPrefix(downloadURL, "https://")

	for _, srv := range vcsPaths {
		if !strings.HasPrefix(downloadPath, srv.prefix) {
			continue
		}
		m := srv.regexp.FindStringSubmatch(downloadPath)
		if m == nil {
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

		// https://community.letsencrypt.org/t/dns-name-has-too-many-labels-error/21577
		match["dnstag"] = strings.ReplaceAll(match["tag"], ".", "-")

		if srv.domain != "" {
			match["domain"] = expand(match, srv.domain)
		}
		if srv.sumPrefix != "" {
			match["sumPrefix"] = expand(match, srv.sumPrefix)
		} else {
			// default to the directory of the file
			match["sumPrefix"] = path.Dir(downloadPath)
		}
		return match, nil
	}
	return nil, errUnknownSite
}
