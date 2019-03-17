package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/certificate-transparency-go/ctutil"
	"github.com/google/certificate-transparency-go/loglist"
	"github.com/google/certificate-transparency-go/x509"
	"github.com/google/certificate-transparency-go/x509util"

	ct "github.com/google/certificate-transparency-go"
)

type logInfoFactory func(*loglist.Log, *http.Client) (*ctutil.LogInfo, error)

// checkChain iterates over any embedded SCTs in the leaf certificate of the chain
// and checks those SCTs.  Returns the counts of valid and invalid embedded SCTs found.
func checkChain(ctx context.Context, lf logInfoFactory, chain []*x509.Certificate, ll *loglist.LogList, hc *http.Client) (int, int) {
	leaf := chain[0]
	if len(leaf.SCTList.SCTList) == 0 {
		return 0, 0
	}

	var issuer *x509.Certificate
	if len(chain) < 2 {
		glog.Info("No issuer in chain; attempting online retrieval")
		var err error
		issuer, err = x509util.GetIssuer(leaf, hc)
		if err != nil {
			fmt.Printf("Failed to get issuer online: %v\n", err)
		}
	} else {
		issuer = chain[1]
	}

	// Build a Merkle leaf that corresponds to the embedded SCTs.  We can use the same
	// leaf for all of the SCTs, as long as the timestamp field gets updated.
	merkleLeaf, err := ct.MerkleTreeLeafForEmbeddedSCT([]*x509.Certificate{leaf, issuer}, 0)
	if err != nil {
		fmt.Printf("Failed to build Merkle leaf: %v\n", err)
		return 0, len(leaf.SCTList.SCTList)
	}

	var valid, invalid int
	for i, sctData := range leaf.SCTList.SCTList {
		subject := fmt.Sprintf("embedded SCT[%d]", i)
		if checkSCT(ctx, lf, subject, merkleLeaf, &sctData, ll, hc) {
			valid++
		} else {
			invalid++
		}
	}
	return valid, invalid
}

// getAndCheckSiteChain retrieves and returns the chain of certificates presented
// for an HTTPS site.  Along the way it checks any external SCTs that are served
// up on the connection alongside the chain.  Returns the chain and counts of
// valid and invalid external SCTs found.
func getAndCheckSiteChain(ctx context.Context, lf logInfoFactory, target string, ll *loglist.LogList, hc *http.Client) ([]*x509.Certificate, int, int, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse URL: %v", err)
	}
	if u.Scheme != "https" {
		return nil, 0, 0, errors.New("non-https URL provided")
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	fmt.Printf("Retrieve certificate chain from TLS connection to %q", host)
	dialer := net.Dialer{Timeout: hc.Timeout}
	conn, err := tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to dial %q: %v", host, err)
	}
	defer conn.Close()

	goChain := conn.ConnectionState().PeerCertificates
	fmt.Printf("Found chain of length %d\n", len(goChain))

	// Convert base crypto/x509.Certificates to our forked x509.Certificate type.
	chain := make([]*x509.Certificate, len(goChain))
	for i, goCert := range goChain {
		cert, err := x509.ParseCertificate(goCert.Raw)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to convert Go Certificate [%d]: %v", i, err)
		}
		chain[i] = cert
	}

	// Check externally-provided SCTs.
	var valid, invalid int
	scts := conn.ConnectionState().SignedCertificateTimestamps
	if len(scts) > 0 {
		merkleLeaf, err := ct.MerkleTreeLeafFromChain(chain, ct.X509LogEntryType, 0 /* timestamp added later */)
		if err != nil {
			fmt.Printf("Failed to build Merkle tree leaf: %v", err)
			return chain, 0, len(scts), nil
		}
		for i, sctData := range scts {
			subject := fmt.Sprintf("external SCT[%d]", i)
			if checkSCT(ctx, lf, subject, merkleLeaf, &x509.SerializedSCT{Val: sctData}, ll, hc) {
				valid++
			} else {
				invalid++
			}

		}
	}

	return chain, valid, invalid, nil
}

// checkSCT performs checks on an SCT and Merkle tree leaf, performing both
// signature validation and online log inclusion checking.  Returns whether
// the SCT is valid.
func checkSCT(ctx context.Context, liFactory logInfoFactory, subject string, merkleLeaf *ct.MerkleTreeLeaf, sctData *x509.SerializedSCT, ll *loglist.LogList, hc *http.Client) bool {
	sct, err := x509util.ExtractSCT(sctData)
	if err != nil {
		fmt.Printf("Failed to deserialize %s data: %v\n", subject, err)
		fmt.Printf("Data: %x\n", sctData.Val)
		return false
	}
	fmt.Printf("Examine %s with timestamp: %d (%v) from logID: %x\n", subject, sct.Timestamp, ct.TimestampToTime(sct.Timestamp), sct.LogID.KeyID[:])
	log := ll.FindLogByKeyHash(sct.LogID.KeyID)
	if log == nil {
		fmt.Printf("Unknown logID: %x, cannot validate %s\n", sct.LogID, subject)
		return false
	}
	logInfo, err := liFactory(log, hc)
	if err != nil {
		fmt.Printf("Failed to build log info for %q log: %v\n", log.Description, err)
		return false
	}

	result := true
	fmt.Printf("Validate %s against log %q...", subject, logInfo.Description)
	if err := logInfo.VerifySCTSignature(*sct, *merkleLeaf); err != nil {
		fmt.Printf("Failed to verify %s signature from log %q: %v\n", subject, log.Description, err)
		result = false
	} else {
		fmt.Printf("Validate %s against log %q... validated\n", subject, log.Description)
	}

	fmt.Printf("Check %s inclusion against log %q...\n", subject, log.Description)
	index, err := logInfo.VerifyInclusion(ctx, *merkleLeaf, sct.Timestamp)
	if err != nil {
		age := time.Since(ct.TimestampToTime(sct.Timestamp))
		if age < logInfo.MMD {
			fmt.Printf("Failed to verify inclusion proof (%v) but %s timestamp is only %v old, less than log's MMD of %d seconds\n", err, subject, age, log.MaximumMergeDelay)
		} else {
			fmt.Printf("Failed to verify inclusion proof for %s: %v\n", subject, err)
		}
		return false
	}
	fmt.Printf("Check %s inclusion against log %q... included at %d\n", subject, log.Description, index)

	return result
}
