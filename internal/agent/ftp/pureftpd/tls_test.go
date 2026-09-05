package pureftpd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func certificateFixture(t *testing.T, expires time.Time) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	value := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), NotBefore: time.Now().Add(-24 * time.Hour), NotAfter: expires,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	cert, err := x509.CreateCertificate(rand.Reader, value, value, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fullchain, privateKey := filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "privkey.pem")
	for path, contents := range map[string][]byte{
		fullchain:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}),
		privateKey: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}),
	} {
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return fullchain, privateKey
}

func TestCertificateLifecycle(t *testing.T) {
	d := driverFixture(t)
	ctx := context.Background()
	cert, key := certificateFixture(t, time.Now().Add(time.Hour))
	calls := 0
	d.run = func(_ context.Context, name string, args ...string) error {
		calls++
		if name != "/usr/bin/systemctl" || len(args) != 2 || args[0] != "restart" || args[1] != "webycp-ftp.service" {
			t.Fatal("unexpected TLS activation command")
		}
		return nil
	}
	for range 2 {
		if err := d.InstallTLS(ctx, cert, key); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("unchanged certificate restarted the service")
	}
	path := filepath.Join(d.dir, "tls.pem")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal("FTP TLS key is not private")
	}
	previous, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"expired", "mismatched", "restart failure"} {
		t.Run(mode, func(t *testing.T) {
			expires := time.Now().Add(2 * time.Hour)
			if mode == "expired" {
				expires = time.Now().Add(-time.Hour)
			}
			nextCert, nextKey := certificateFixture(t, expires)
			if mode == "mismatched" {
				nextKey = key
			}
			restarts := 0
			d.run = func(context.Context, string, ...string) error {
				restarts++
				if restarts == 1 {
					return errors.New("restart failed")
				}
				return nil
			}
			if err := d.InstallTLS(ctx, nextCert, nextKey); err == nil {
				t.Fatal("invalid certificate change succeeded")
			}
			current, err := os.ReadFile(path)
			if err != nil || string(current) != string(previous) {
				t.Fatal("failed certificate update replaced the working certificate")
			}
			if (mode == "restart failure" && restarts != 2) || (mode != "restart failure" && restarts != 0) {
				t.Fatal("certificate validation or recovery order is incorrect")
			}
		})
	}
}
