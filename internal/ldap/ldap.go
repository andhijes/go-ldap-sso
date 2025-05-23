package ldapauth

import (
	"crypto/tls"
	"fmt"
	"go-ldap-sso/config"
	"log"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
)

type LDAPClient struct {
	Config    *config.LDAPConfig
	conn      *ldap.Conn
	connMutex sync.Mutex
	isClosed  bool
}

func NewLDAPClient(cfg *config.LDAPConfig) (*LDAPClient, error) {
	client := &LDAPClient{
		Config:   cfg,
		isClosed: false,
	}

	if err := client.ensureConnection(); err != nil {
		return nil, err
	}

	return client, nil
}

func (lc *LDAPClient) ensureConnection() error {
	lc.connMutex.Lock()
	defer lc.connMutex.Unlock()

	if lc.conn != nil && !lc.isClosed {
		// check connection with empty search
		searchRequest := ldap.NewSearchRequest(
			lc.Config.BaseDN,
			ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			[]string{"1.1"},
			nil,
		)

		_, err := lc.conn.Search(searchRequest)
		if err == nil {
			return nil
		}
		lc.conn.Close()
	}

	var conn *ldap.Conn
	var err error

	server := fmt.Sprintf("%s:%d", lc.Config.Host, lc.Config.Port)

	if lc.Config.UseSSL {
		conn, err = ldap.DialTLS("tcp", server, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.Dial("tcp", server)
	}

	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}

	// Set timeout
	conn.SetTimeout(10 * time.Second)

	err = conn.Bind(lc.Config.BindDN, lc.Config.BindPass)
	if err != nil {
		conn.Close()
		return fmt.Errorf("admin bind failed: %v", err)
	}

	lc.conn = conn
	lc.isClosed = false
	return nil
}

func (lc *LDAPClient) Authenticate(username, password string) (string, error) {
	if err := lc.ensureConnection(); err != nil {
		return "", err
	}

	lc.connMutex.Lock()
	defer lc.connMutex.Unlock()

	// Search user
	searchRequest := ldap.NewSearchRequest(
		lc.Config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)),
		[]string{"dn", "cn", "mail"},
		nil,
	)

	sr, err := lc.conn.Search(searchRequest)
	if err != nil {
		if err := lc.ensureConnection(); err != nil {
			return "", fmt.Errorf("search failed: %v", err)
		}
		sr, err = lc.conn.Search(searchRequest)
		if err != nil {
			return "", fmt.Errorf("search failed after retry: %v", err)
		}
	}

	if len(sr.Entries) != 1 {
		return "", fmt.Errorf("user not found or duplicate entries")
	}

	userDN := sr.Entries[0].DN

	// Verify credentials
	err = lc.conn.Bind(userDN, password)
	if err != nil {
		return "", fmt.Errorf("invalid credentials: %v", err)
	}

	if err := lc.conn.Bind(lc.Config.BindDN, lc.Config.BindPass); err != nil {
		log.Printf("Warning: failed to rebind as admin: %v", err)
	}

	return sr.Entries[0].GetAttributeValue("mail"), nil
}

func (lc *LDAPClient) Close() {
	lc.connMutex.Lock()
	defer lc.connMutex.Unlock()

	if lc.conn != nil && !lc.isClosed {
		lc.conn.Close()
		lc.isClosed = true
	}
}

func (lc *LDAPClient) HealthCheck() error {
	lc.connMutex.Lock()
	defer lc.connMutex.Unlock()

	if lc.conn == nil || lc.isClosed {
		return fmt.Errorf("no active connection")
	}

	searchRequest := ldap.NewSearchRequest(
		lc.Config.BaseDN,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"1.1"},
		nil,
	)

	_, err := lc.conn.Search(searchRequest)
	return err
}
