// Package certcheck validates a TLS certificate and private key pair
// before they are embedded into the platform configuration file.
//
// Checks performed:
//   - PEM decoding and certificate parsing
//   - Expiry (hard error if expired, warning if < 30 days)
//   - FQDN match against SANs and Common Name (wildcard-aware)
//   - Certificate / private key pair consistency
//   - RSA public key cross-check between cert and key
package certcheck

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// Result holds the outcome of a certificate validation run.
type Result struct {
	Subject   string
	Issuer    string
	NotBefore time.Time
	NotAfter  time.Time
	DNSNames  []string
	IsValid   bool
	Warnings  []string
	Errors    []string
}

// Check validates certPEM and keyPEM against the target fqdn.
// It returns a Result regardless of validity so the caller can
// decide whether to proceed or abort.
func Check(certPEM, keyPEM, fqdn string) (*Result, error) {
	r := &Result{}

	// ── Parse certificate ─────────────────────────────────────────────────
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	r.Subject = cert.Subject.CommonName
	r.Issuer = cert.Issuer.CommonName
	r.NotBefore = cert.NotBefore
	r.NotAfter = cert.NotAfter
	r.DNSNames = cert.DNSNames

	// ── Expiry ────────────────────────────────────────────────────────────
	now := time.Now()
	switch {
	case now.Before(cert.NotBefore):
		r.Errors = append(r.Errors, "certificate is not yet valid")
	case now.After(cert.NotAfter):
		r.Errors = append(r.Errors,
			fmt.Sprintf("certificate expired on %s", cert.NotAfter.Format("2006-01-02")))
	default:
		daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
		if daysLeft < 30 {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("certificate expires in %d days (%s)",
					daysLeft, cert.NotAfter.Format("2006-01-02")))
		}
	}

	// ── FQDN match ────────────────────────────────────────────────────────
	matched := matchDomain(cert.Subject.CommonName, fqdn)
	for _, san := range cert.DNSNames {
		if matchDomain(san, fqdn) {
			matched = true
			break
		}
	}
	if !matched {
		r.Errors = append(r.Errors,
			fmt.Sprintf("FQDN %q does not match certificate (SANs: %s)",
				fqdn, strings.Join(cert.DNSNames, ", ")))
	}

	// ── Key pair consistency ──────────────────────────────────────────────
	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		r.Errors = append(r.Errors,
			fmt.Sprintf("certificate and key do not match: %v", err))
	} else {
		// Additional RSA public-key cross-check.
		block2, _ := pem.Decode([]byte(keyPEM))
		if block2 != nil {
			key, err := x509.ParsePKCS8PrivateKey(block2.Bytes)
			if err == nil {
				if rsaKey, ok := key.(*rsa.PrivateKey); ok {
					if !rsaKey.PublicKey.Equal(cert.PublicKey) {
						r.Errors = append(r.Errors,
							"private key does not match certificate public key")
					}
				}
			}
		}
	}

	r.IsValid = len(r.Errors) == 0
	return r, nil
}

// matchDomain checks pattern (possibly wildcard) against host.
// A wildcard pattern like *.example.com matches one subdomain level only.
func matchDomain(pattern, host string) bool {
	if pattern == host {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // e.g. ".example.com"
		if strings.HasSuffix(host, suffix) {
			prefix := strings.TrimSuffix(host, suffix)
			return !strings.Contains(prefix, ".")
		}
	}
	return false
}

// Print writes a human-readable summary of the Result to stdout.
func Print(r *Result) {
	fmt.Printf("\n=== Certificate Check ===\n")
	fmt.Printf("Subject    : %s\n", r.Subject)
	fmt.Printf("Issuer     : %s\n", r.Issuer)
	fmt.Printf("Valid from : %s\n", r.NotBefore.Format("2006-01-02"))
	fmt.Printf("Valid to   : %s\n", r.NotAfter.Format("2006-01-02"))
	fmt.Printf("SANs       : %s\n", strings.Join(r.DNSNames, ", "))

	for _, w := range r.Warnings {
		fmt.Printf("⚠  %s\n", w)
	}
	for _, e := range r.Errors {
		fmt.Printf("✗  %s\n", e)
	}
	if r.IsValid {
		fmt.Println("✓  Certificate is valid")
	}
}