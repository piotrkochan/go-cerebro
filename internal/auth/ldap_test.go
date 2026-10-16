package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/lmenezes/cerebro/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDAPService_RequiresLDAPSUnlessExplicitlyInsecure(t *testing.T) {
	_, err := NewLDAPService(config.LDAPAuth{
		URL:          "ldap://ldap.example:389",
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires ldaps")
}

func TestNewLDAPService_RejectsInvalidCAFile(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ldap-ca.pem")
	require.NoError(t, os.WriteFile(caFile, []byte("not a pem"), 0o600))

	_, err := NewLDAPService(config.LDAPAuth{
		URL:          "ldaps://ldap.example:636",
		CACertFile:   caFile,
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid PEM certificate")
}

func TestNewLDAPService_ConfiguresRequiredGroups(t *testing.T) {
	service, err := NewLDAPService(config.LDAPAuth{
		URL:            "ldap://ldap.example:389",
		BaseDN:         "dc=example,dc=org",
		UserTemplate:   "uid=%s,%s",
		InsecureLDAP:   true,
		RequiredGroups: []string{"cerebro-admins"},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"cerebro-admins"}, service.requiredGroups)
	assert.True(t, hasAnyGroup([]string{"users", "cerebro-admins"}, service.requiredGroups))
	assert.False(t, hasAnyGroup([]string{"users"}, service.requiredGroups))
	assert.Equal(t, 5*time.Second, service.dialTimeout)
	assert.Equal(t, 30*time.Second, service.readTimeout)
}

func TestNewLDAPService_ConfiguresTimeouts(t *testing.T) {
	service, err := NewLDAPService(config.LDAPAuth{
		URL:                "ldaps://ldap.example:636",
		BaseDN:             "dc=example,dc=org",
		UserTemplate:       "uid=%s,%s",
		DialTimeoutSeconds: 2,
		ReadTimeoutSeconds: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, service.dialTimeout)
	assert.Equal(t, 3*time.Second, service.readTimeout)
}

func TestLDAPAuthenticate_ReadTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		client, err := listener.Accept()
		if err != nil {
			return
		}
		defer client.Close()
		if _, err := ber.ReadPacket(client); err != nil {
			return
		}
		<-done // Deliberately leave the bind request unanswered.
	}()

	service, err := NewLDAPService(config.LDAPAuth{
		URL:                "ldap://" + listener.Addr().String(),
		BaseDN:             "dc=example,dc=org",
		UserTemplate:       "uid=%s,%s",
		InsecureLDAP:       true,
		ReadTimeoutSeconds: 1,
	})
	require.NoError(t, err)
	start := time.Now()
	_, err = service.Authenticate("alice", "password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Less(t, time.Since(start), 3*time.Second)
}

func TestNewLDAPService_RejectsMixedAndInsecureURLs(t *testing.T) {
	settings := config.LDAPAuth{
		URL:          "ldaps://dc1.example.org:636",
		URLs:         []string{"ldaps://dc2.example.org:636"},
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	}
	_, err := NewLDAPService(settings)
	require.ErrorContains(t, err, "mutually exclusive")

	settings.URL = ""
	settings.URLs = []string{"ldaps://dc1.example.org:636", "ldap://dc2.example.org:389"}
	_, err = NewLDAPService(settings)
	require.ErrorContains(t, err, "requires ldaps")
}

func TestLDAPDial_FailsOverAfterTLSHandshakeError(t *testing.T) {
	bad := httptest.NewServer(nil)
	defer bad.Close()
	good := httptest.NewTLSServer(nil)
	defer good.Close()

	pool := x509.NewCertPool()
	pool.AddCert(good.Certificate())
	service := &LDAPService{
		urls: []string{
			"ldaps://" + bad.Listener.Addr().String(),
			"ldaps://" + good.Listener.Addr().String(),
		},
		tlsConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12, ServerName: "wrong.example.org"},
	}
	conn, err := service.dial()
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestLDAPAuthenticate_DoesNotFailOverAfterInvalidCredentials(t *testing.T) {
	first, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer first.Close()
	second, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer second.Close()

	binds := make(chan struct{}, 1)
	go func() {
		client, err := first.Accept()
		if err != nil {
			return
		}
		defer client.Close()
		_ = client.SetDeadline(time.Now().Add(time.Second))
		request, err := ber.ReadPacket(client)
		if err != nil {
			return
		}
		binds <- struct{}{}
		response := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAP Response")
		response.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, request.Children[0].Value, "MessageID"))
		result := ber.Encode(ber.ClassApplication, ber.TypeConstructed, 1, nil, "Bind Response")
		result.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, 49, "Invalid credentials"))
		result.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", "Matched DN"))
		result.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", "Diagnostic message"))
		response.AppendChild(result)
		_, _ = client.Write(response.Bytes())
	}()

	service, err := NewLDAPService(config.LDAPAuth{
		URLs:         []string{"ldap://" + first.Addr().String(), "ldap://" + second.Addr().String()},
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
		InsecureLDAP: true,
	})
	require.NoError(t, err)
	_, err = service.Authenticate("alice", "wrong")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.Len(t, binds, 1)

	_ = second.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond))
	unexpected, err := second.Accept()
	if err == nil {
		unexpected.Close()
		t.Fatal("second LDAP server received a connection after invalid credentials")
	}
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	require.True(t, netErr.Timeout(), fmt.Sprintf("unexpected accept error: %v", err))
}
