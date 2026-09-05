package pureftpd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GVALFER/WEBYCP/internal/agent/configfile"
)

// InstallTLS accepts certificate lifecycle paths, never an untrusted request's
// paths. The combined PEM is readable only by the privileged daemon.
func (d *Driver) InstallTLS(ctx context.Context, fullchain, privateKey string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cert, err := os.ReadFile(fullchain)
	if err != nil {
		return fmt.Errorf("read FTP certificate: %w", err)
	}
	key, err := os.ReadFile(privateKey)
	if err != nil {
		return fmt.Errorf("read FTP private key: %w", err)
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("FTP certificate and private key are invalid")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil || time.Now().Before(leaf.NotBefore) || !time.Now().Before(leaf.NotAfter) {
		return fmt.Errorf("FTP certificate is not currently valid")
	}
	if err := secureDir(d.dir); err != nil {
		return err
	}
	path := filepath.Join(d.dir, "tls.pem")
	previous, err := configfile.Take(path)
	if err != nil {
		return err
	}
	contents := append(append(append([]byte{}, cert...), '\n'), key...)
	if previous.Exists && bytes.Equal(previous.Data, contents) && previous.Mode == 0o600 {
		return nil
	}
	if err := configfile.Write(path, contents, 0o600); err != nil {
		return err
	}
	if err := d.run(ctx, "/usr/bin/systemctl", "restart", "webycp-ftp.service"); err != nil {
		failure := errors.Join(fmt.Errorf("activate FTP certificate"), previous.Restore())
		if previous.Exists {
			if err := d.run(ctx, "/usr/bin/systemctl", "restart", "webycp-ftp.service"); err != nil {
				failure = errors.Join(failure, fmt.Errorf("restart FTP with previous certificate"))
			}
		}
		return failure
	}
	return nil
}
