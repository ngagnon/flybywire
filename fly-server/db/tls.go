package db

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"time"
)

func (db *Handle) loadTlsCert() {
	if db.err != nil {
		return
	}

	if !certExists(db) {
		db.generateTlsCert()
	}

	expiry, err := getCertExpiry(db)

	if err != nil {
		db.err = err
		return
	}

	// Regenerate the certificate if it's expired or about to expire (1h)
	if expiry.Before(time.Now().Add(1 * time.Hour)) {
		expiry = time.Now().Add(365 * 24 * time.Hour)
		db.generateTlsCert()
	}

	t := time.NewTimer(expiry.Sub(time.Now().Add(1 * time.Hour)))

	go (func() {
		<-t.C
		db.generateTlsCert()
		db.loadTlsCert()
	})()

	certPath := path.Join(db.dir, ".fly/cert.pem")
	keyPath := path.Join(db.dir, ".fly/key.pem")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)

	if err != nil {
		db.err = fmt.Errorf("Failed to load TLS certificate: %v", err)
		return
	}

	db.certLock.Lock()
	db.cert = &cert
	db.certLock.Unlock()
}

func certExists(db *Handle) bool {
	certPath := path.Join(db.dir, ".fly/cert.pem")
	_, err := os.Stat(certPath)
	return !errors.Is(err, os.ErrNotExist)
}

func getCertExpiry(db *Handle) (time.Time, error) {
	certPath := path.Join(db.dir, ".fly/cert.pem")
	pemBytes, err := os.ReadFile(certPath)

	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to get TLS certificate expiry: failed to read certificate: %v", err)
	}

	block, _ := pem.Decode(pemBytes)

	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("Failed to get TLS certificate expiry: failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)

	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to get TLS certificate expiry: failed to parse certificate: %v", err)
	}

	return cert.NotAfter, nil
}

func (db *Handle) generateTlsCert() {
	generateCert(db, 365*24*time.Hour)
}

func generateCert(db *Handle, expiry time.Duration) {
	if db.err != nil {
		return
	}

	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	priv, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to generate private key: %v", err)
		return
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to generate serial number: %v", err)
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(expiry),
		KeyUsage:    keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: %v", err)
		return
	}

	certPath := path.Join(db.dir, ".fly/cert.pem")
	certOut, err := os.Create(certPath)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to open cert.pem for writing: %v", err)
		return
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to write data to cert.pem: %v", err)
		return
	}

	certOut.Close()

	keyPath := path.Join(db.dir, ".fly/key.pem")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to open key.pem for writing: %v", err)
		return
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)

	if err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: unable to marshal private key: %v", err)
		return
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		db.err = fmt.Errorf("Failed to generate TLS certificate: failed to write data to key.pem: %v", err)
		return
	}

	keyOut.Close()
}
