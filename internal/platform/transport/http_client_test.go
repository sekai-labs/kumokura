package transport_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/platform/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPClient(t *testing.T) {
	opts := transport.DefaultClientOptions()
	client, err := transport.NewHTTPClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, 120*time.Second, client.Timeout)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCustomCACertPath(t *testing.T) {
	tmp := t.TempDir()
	certPath := filepath.Join(tmp, "ca.pem")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Kumokura Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))

	opts := transport.DefaultClientOptions()
	opts.CustomCACertPath = certPath
	client, err := transport.NewHTTPClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)

	optsInvalid := transport.DefaultClientOptions()
	optsInvalid.CustomCACertPath = filepath.Join(tmp, "nonexistent.pem")
	_, err = transport.NewHTTPClient(optsInvalid)
	assert.Error(t, err)
}
