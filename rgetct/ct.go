package rgetct

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

// CheckX509 iterates over any X509 extension SCTs in the leaf certificate of the chain
// and checks those SCTs.  Returns the counts of valid and invalid embedded SCTs found.
func CheckX509(ctx context.Context, lf logInfoFactory, chain []*x509.Certificate, ll *loglist.LogList, hc *http.Client) (valid int, invalid int, logs []loglist.Log) {
	leaf := chain[0]
	if len(leaf.SCTList.SCTList) == 0 {
		return
	}

	var issuer *x509.Certificate
	if len(chain) < 2 {
		glog.Info("No issuer in chain; attempting online retrieval")
		var err error
		issuer, err = x509util.GetIssuer(leaf, hc)
		if err != nil {
			fmt.Printf("Failed to get issuer online: %v\n", err)
			return
		}
	} else {
		issuer = chain[1]
	}

	// Build a Merkle leaf that corresponds to the embedded SCTs.  We can use the same
	// leaf for all of the SCTs, as long as the timestamp field gets updated.
	merkleLeaf, err := ct.MerkleTreeLeafForEmbeddedSCT([]*x509.Certificate{leaf, issuer}, 0)
	if err != nil {
		fmt.Printf("Failed to build Merkle leaf: %v\n", err)
		invalid = len(leaf.SCTList.SCTList)
		return
	}

	for i, sctData := range leaf.SCTList.SCTList {
		subject := fmt.Sprintf("embedded SCT[%d]", i)
		ok, log := checkSCT(ctx, lf, subject, merkleLeaf, &sctData, ll, hc)
		logs = append(logs, *log)
		if ok {
			valid++
		} else {
			invalid++
		}
	}
	return
}

// GetSiteSCTs retrieves and returns the x509 chain and TLS SCTs presented
// for an HTTPS site.
func GetSiteSCTs(ctx context.Context, target string, hc *http.Client) (chain []*x509.Certificate, tlsSCTs [][]byte, err error) {
	u, err := url.Parse(target)
	if err != nil {
		err = fmt.Errorf("failed to parse URL: %v", err)
		return
	}
	if u.Scheme != "https" {
		err = errors.New("non-https URL provided")
		return
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	dialer := net.Dialer{Timeout: hc.Timeout}
	conn, err := tls.DialWithDialer(&dialer, "tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		err = fmt.Errorf("failed to dial %q: %v", host, err)
		return
	}
	defer conn.Close()

	goChain := conn.ConnectionState().PeerCertificates

	// Convert base crypto/x509.Certificates to our forked x509.Certificate type.
	chain = make([]*x509.Certificate, len(goChain))
	var verifiedHostname bool
	for i, goCert := range goChain {
		var cert *x509.Certificate
		cert, err = x509.ParseCertificate(goCert.Raw)
		if err != nil {
			err = fmt.Errorf("failed to convert Go Certificate [%d]: %v", i, err)
			return
		}

		if err := cert.VerifyHostname(u.Host); err == nil {
			verifiedHostname = true
		}

		chain[i] = cert
	}

	if verifiedHostname == false {
		err = errors.New("cannot verify host for target")
		return
	}

	// Check externally-provided SCTs.
	tlsSCTs = conn.ConnectionState().SignedCertificateTimestamps

	return
}

// CheckTLS iterates over any TLS extension SCTs presented from a connection
// and checks those SCTs.  Returns the counts of valid and invalid
// SCTs found.
func CheckTLS(ctx context.Context, scts [][]byte, chain []*x509.Certificate, lf logInfoFactory, target string, ll *loglist.LogList, hc *http.Client) (valid int, invalid int, logs []loglist.Log) {
	if len(scts) > 0 {
		var merkleLeaf *ct.MerkleTreeLeaf
		merkleLeaf, err := ct.MerkleTreeLeafFromChain(chain, ct.X509LogEntryType, 0 /* timestamp added later */)
		if err != nil {
			fmt.Printf("Failed to build Merkle tree leaf: %v\n", err)
			invalid = len(scts)
			return
		}
		for i, sctData := range scts {
			subject := fmt.Sprintf("external SCT[%d]", i)
			ok, log := checkSCT(ctx, lf, subject, merkleLeaf, &x509.SerializedSCT{Val: sctData}, ll, hc)
			logs = append(logs, *log)
			if ok {
				valid++
			} else {
				invalid++
			}

		}
	}

	return
}

// checkSCT performs checks on an SCT and Merkle tree leaf, performing both
// signature validation and online log inclusion checking.  Returns whether
// the SCT is valid.
func checkSCT(ctx context.Context, liFactory logInfoFactory, subject string, merkleLeaf *ct.MerkleTreeLeaf, sctData *x509.SerializedSCT, ll *loglist.LogList, hc *http.Client) (result bool, log *loglist.Log) {
	sct, err := x509util.ExtractSCT(sctData)
	if err != nil {
		fmt.Printf("Failed to deserialize %s data: %v\n", subject, err)
		fmt.Printf("Data: %x\n", sctData.Val)
		return
	}

	// TODO(philips): add verbose logging
	// fmt.Printf("Examine %s with timestamp: %d (%v) from logID: %x\n", subject, sct.Timestamp, ct.TimestampToTime(sct.Timestamp), sct.LogID.KeyID[:])
	log = ll.FindLogByKeyHash(sct.LogID.KeyID)
	if log == nil {
		fmt.Printf("Unknown logID: %x, cannot validate %s\n", sct.LogID, subject)
		return
	}
	logInfo, err := liFactory(log, hc)
	if err != nil {
		fmt.Printf("Failed to build log info for %q log: %v\n", log.Description, err)
		return
	}

	result = true
	if err := logInfo.VerifySCTSignature(*sct, *merkleLeaf); err != nil {
		fmt.Printf("Failed to verify %s signature from log %q: %v\n", subject, log.Description, err)
		result = false
	}

	_, err = logInfo.VerifyInclusion(ctx, *merkleLeaf, sct.Timestamp)
	if err != nil {
		age := time.Since(ct.TimestampToTime(sct.Timestamp))
		if age < logInfo.MMD {
			fmt.Printf("Failed to verify inclusion proof (%v) but %s timestamp is only %v old, less than log's MMD of %d seconds\n", err, subject, age, log.MaximumMergeDelay)
			// TODO(philips): fix this case.
			result = true
			return
		} else {
			fmt.Printf("Failed to verify inclusion proof for %s: %v\n", subject, err)
		}
		result = false
		return
	}

	return
}
