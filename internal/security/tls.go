package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	tlsCertFile = ".indiekku_cert.pem"
	tlsKeyFile  = ".indiekku_key.pem"
)

// EnsureTLSCert loads the persisted TLS cert/key from disk, generating and
// saving them if they don't exist yet. extraIPs are added as SANs on first gen.
func EnsureTLSCert(extraIPs ...string) (tls.Certificate, error) {
	certPath := filepath.Join(".", tlsCertFile)
	keyPath := filepath.Join(".", tlsKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		return tls.X509KeyPair(certPEM, keyPEM)
	}

	// Generate a new cert and persist it
	cert, certPEM, keyPEM, err := generateSelfSignedCertPEM(extraIPs...)
	if err != nil {
		return tls.Certificate{}, err
	}
	_ = os.WriteFile(certPath, certPEM, 0600)
	_ = os.WriteFile(keyPath, keyPEM, 0600)
	return cert, nil
}

// GenerateSelfSignedCert creates an in-memory ECDSA self-signed TLS certificate
// valid for 10 years. localhost and 127.0.0.1 are always included; any extra
// IPs passed (e.g. the server's public IP) are added as SANs.
func GenerateSelfSignedCert(extraIPs ...string) (tls.Certificate, error) {
	cert, _, _, err := generateSelfSignedCertPEM(extraIPs...)
	return cert, err
}

// generateSelfSignedCertPEM is the internal implementation that returns both
// the parsed certificate and the raw PEM bytes for persistence.
func generateSelfSignedCertPEM(extraIPs ...string) (tls.Certificate, []byte, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	ipAddresses := []net.IP{net.ParseIP("127.0.0.1")}
	for _, ip := range extraIPs {
		if parsed := net.ParseIP(ip); parsed != nil {
			ipAddresses = append(ipAddresses, parsed)
		}
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"indiekku"},
			CommonName:   "indiekku",
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           ipAddresses,
		DNSNames:              []string{"localhost"},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return cert, certPEM, keyPEM, err
}
