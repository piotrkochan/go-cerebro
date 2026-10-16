package auth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		if _, _, err := readLDAPOperation(client); err != nil {
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

func TestNewLDAPService_ValidatesStartTLS(t *testing.T) {
	settings := config.LDAPAuth{BaseDN: "dc=example,dc=org", UserTemplate: "uid=%s,%s", StartTLS: true}
	settings.URL = "ldaps://dc.example.org:636"
	_, err := NewLDAPService(settings)
	require.ErrorContains(t, err, "start_tls requires ldap://")

	settings.URL = "ldap://dc.example.org:389"
	settings.InsecureLDAP = true
	_, err = NewLDAPService(settings)
	require.ErrorContains(t, err, "mutually exclusive")

	settings.InsecureLDAP = false
	settings.URL = ""
	settings.URLs = []string{"ldap://dc1.example.org:389", "ldaps://dc2.example.org:636"}
	_, err = NewLDAPService(settings)
	require.ErrorContains(t, err, "start_tls requires ldap://")
}

func TestLDAPAuthenticate_StartTLSFailsOverBeforeBind(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	defer certServer.Close()
	cert := certServer.TLS.Certificates[0]
	caFile := filepath.Join(t.TempDir(), "ldap-ca.pem")
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o600))

	startServer := func(resultCode int) (string, <-chan int) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { listener.Close() })
		operations := make(chan int, 2)
		go func() {
			defer close(operations)
			client, err := listener.Accept()
			if err != nil {
				return
			}
			defer client.Close()
			_ = client.SetDeadline(time.Now().Add(3 * time.Second))
			messageID, operation, err := readLDAPOperation(client)
			if err != nil {
				return
			}
			operations <- operation
			writeLDAPResult(client, messageID, 24, resultCode)
			if resultCode != 0 {
				return
			}
			secure := tls.Server(client, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
			if err := secure.Handshake(); err != nil {
				return
			}
			messageID, operation, err = readLDAPOperation(secure)
			if err != nil {
				return
			}
			operations <- operation
			writeLDAPResult(secure, messageID, 1, 0)
		}()
		return "ldap://" + listener.Addr().String(), operations
	}
	firstURL, firstOps := startServer(52) // Unavailable: no plaintext bind is allowed.
	secondURL, secondOps := startServer(0)
	service, err := NewLDAPService(config.LDAPAuth{
		URLs:         []string{firstURL, secondURL},
		StartTLS:     true,
		CACertFile:   caFile,
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})
	require.NoError(t, err)
	identity, err := service.Authenticate("alice", "password")
	require.NoError(t, err)
	assert.Equal(t, "alice", identity.Username)
	assert.Equal(t, []int{23}, collectLDAPOperations(firstOps))
	assert.Equal(t, []int{23, 0}, collectLDAPOperations(secondOps))
}

func TestLDAPAuthenticate_StartTLSRejectsUntrustedCertificate(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	defer certServer.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	bindSeen := make(chan bool, 1)
	go func() {
		client, err := listener.Accept()
		if err != nil {
			bindSeen <- false
			return
		}
		defer client.Close()
		_ = client.SetDeadline(time.Now().Add(3 * time.Second))
		messageID, _, err := readLDAPOperation(client)
		if err != nil {
			bindSeen <- false
			return
		}
		writeLDAPResult(client, messageID, 24, 0)
		secure := tls.Server(client, &tls.Config{Certificates: certServer.TLS.Certificates, MinVersion: tls.VersionTLS12})
		if err := secure.Handshake(); err != nil {
			bindSeen <- false
			return
		}
		_, _, err = readLDAPOperation(secure)
		bindSeen <- err == nil
	}()
	service, err := NewLDAPService(config.LDAPAuth{
		URL:          "ldap://" + listener.Addr().String(),
		StartTLS:     true,
		BaseDN:       "dc=example,dc=org",
		UserTemplate: "uid=%s,%s",
	})
	require.NoError(t, err)
	_, err = service.Authenticate("alice", "password")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.False(t, <-bindSeen)
}

func TestLDAPDial_StartTLSHandshakeTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		client, err := listener.Accept()
		if err != nil {
			return
		}
		defer client.Close()
		messageID, _, err := readLDAPOperation(client)
		if err != nil {
			return
		}
		writeLDAPResult(client, messageID, 24, 0)
		// Leave the TLS handshake unanswered until the client's deadline closes it.
		_, _ = io.Copy(io.Discard, client)
	}()
	service, err := NewLDAPService(config.LDAPAuth{
		URL:                "ldap://" + listener.Addr().String(),
		StartTLS:           true,
		DialTimeoutSeconds: 1,
		BaseDN:             "dc=example,dc=org",
		UserTemplate:       "uid=%s,%s",
	})
	require.NoError(t, err)
	start := time.Now()
	_, err = service.dial()
	require.Error(t, err)
	assert.Less(t, time.Since(start), 3*time.Second)
}

func readLDAPOperation(conn net.Conn) (int, int, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, 0, err
	}
	if header[0] != 0x30 {
		return 0, 0, fmt.Errorf("unexpected LDAP envelope tag: %x", header[0])
	}
	length := int(header[1])
	if length&0x80 != 0 {
		count := length & 0x7f
		if count == 0 || count > 4 {
			return 0, 0, fmt.Errorf("invalid LDAP envelope length")
		}
		extra := make([]byte, count)
		if _, err := io.ReadFull(conn, extra); err != nil {
			return 0, 0, err
		}
		header = append(header, extra...)
		length = 0
		for _, b := range extra {
			length = length<<8 | int(b)
		}
	}
	if length > 1<<20 {
		return 0, 0, fmt.Errorf("LDAP test request exceeds limit")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, 0, err
	}
	var envelope, operation asn1.RawValue
	if _, err := asn1.Unmarshal(append(header, payload...), &envelope); err != nil {
		return 0, 0, err
	}
	var messageID int
	rest, err := asn1.Unmarshal(envelope.Bytes, &messageID)
	if err != nil {
		return 0, 0, err
	}
	if _, err := asn1.Unmarshal(rest, &operation); err != nil {
		return 0, 0, err
	}
	return messageID, operation.Tag, nil
}

func writeLDAPResult(conn net.Conn, messageID, tag, resultCode int) {
	result := []byte{0x0a, 0x01, byte(resultCode), 0x04, 0x00, 0x04, 0x00}
	response, err := asn1.Marshal(struct {
		MessageID int
		Operation asn1.RawValue
	}{messageID, asn1.RawValue{Class: asn1.ClassApplication, Tag: tag, IsCompound: true, Bytes: result}})
	if err == nil {
		_, _ = conn.Write(response)
	}
}

func collectLDAPOperations(operations <-chan int) []int {
	var result []int
	for op := range operations {
		result = append(result, op)
	}
	return result
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
		messageID, _, err := readLDAPOperation(client)
		if err != nil {
			return
		}
		binds <- struct{}{}
		writeLDAPResult(client, messageID, 1, 49)
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
