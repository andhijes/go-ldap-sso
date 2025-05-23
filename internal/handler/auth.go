package handler

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"go-ldap-sso/config"
	"go-ldap-sso/db"
	"go-ldap-sso/internal/auth"
	"go-ldap-sso/internal/helper"
	ldapauth "go-ldap-sso/internal/ldap"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gorilla/sessions"
)

type AuthHandler struct {
	cfg        *config.Config
	samlSP     *samlsp.Middleware
	store      sessions.Store
	ldapClient *ldapauth.LDAPClient
	db         *db.Database
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRes struct {
	Token string `json:"token"`
}

func NewAuthHandler(cfg *config.Config, db *db.Database) (*AuthHandler, error) {
	// 1️⃣ Session store with secret from config (rotate via env/config)
	store := sessions.NewCookieStore([]byte("cfg.AuthConfig.SessionSecret"))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(1 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   false, // for localhost/http
		SameSite: http.SameSiteLaxMode,
	}

	// 2️⃣ Initialize SAML SP (with Secure=false for local dev)
	samlSP, err := setupSAML(cfg)
	if err != nil {
		return nil, fmt.Errorf("SAML init failed: %w", err)
	}
	log.Println("✅ SAML SP initialized:")
	log.Printf("   • EntityID: %s", samlSP.ServiceProvider.Metadata().EntityID)
	log.Printf("   • ACS URL:  %s", samlSP.ServiceProvider.AcsURL.String())

	// 3️⃣ Initialize LDAP client (no defer here!)
	ldapClient, err := ldapauth.NewLDAPClient(&cfg.LDAPConfig)
	if err != nil {
		return nil, fmt.Errorf("LDAP init failed: %w", err)
	}
	// Health-check loop—client will stay open until you call ldapClient.Close()
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := ldapClient.HealthCheck(); err != nil {
				log.Printf("⚠️ LDAP unhealthy: %v", err)
			}
		}
	}()

	return &AuthHandler{
		cfg:        cfg,
		samlSP:     samlSP,
		store:      store,
		ldapClient: ldapClient,
		db:         db,
	}, nil
}

func setupSAML(cfg *config.Config) (*samlsp.Middleware, error) {
	// 1. Load certificate
	certPEM, err := os.ReadFile(cfg.SAMLConfig.CertFile)
	if err != nil {
		return nil, fmt.Errorf("read cert file: %w", err)
	}

	// 2. Load private key
	keyPEM, err := os.ReadFile(cfg.SAMLConfig.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	// 3. Decode PEM blocks
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing certificate")
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing private key")
	}

	// 4. Parse certificate
	leafCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w (verify your cert is in PEM format)", err)
	}

	// 5. Parse private key (supports both PKCS1 and PKCS8)
	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 if PKCS1 fails
		key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key (neither PKCS1 nor PKCS8 format): %w", err)
		}
		privateKey = key.(*rsa.PrivateKey)
	}

	// Build service provider root URL
	rootURL, err := url.Parse(fmt.Sprintf("http://localhost:%s", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("parse root URL: %w", err)
	}

	// Fetch IdP metadata
	idpURL, err := url.Parse(cfg.SAMLConfig.IDPMetadata)
	if err != nil {
		return nil, fmt.Errorf("parse IdP metadata URL: %w", err)
	}

	log.Println("🔄 Fetching IdP metadata from", idpURL)

	idpMetadata, err := samlsp.FetchMetadata(
		context.Background(),
		http.DefaultClient,
		*helper.MustParseURL(cfg.SAMLConfig.IDPMetadata),
	)
	if err != nil {
		return nil, err
	}

	opts := samlsp.Options{
		URL:               *rootURL,
		Key:               privateKey,
		Certificate:       leafCert,
		IDPMetadata:       idpMetadata,
		CookieName:        "saml_token",
		CookieSameSite:    http.SameSiteLaxMode, // 🔁 change to SameSiteNoneMode + HTTPS if needed
		AllowIDPInitiated: true,
	}

	// Initialize Service Provider middleware
	sp, err := samlsp.New(opts)
	if err != nil {
		return nil, fmt.Errorf("create SAML SP: %w", err)
	}

	log.Printf("→ SAML ACS Path: %q", sp.ServiceProvider.AcsURL.Path)

	log.Println("✅ SAML SP initialized")
	log.Printf("🔐 EntityID: %s", sp.ServiceProvider.Metadata().EntityID)
	log.Printf("➡️  ACS URL: %s", sp.ServiceProvider.AcsURL.String())

	return sp, nil
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Render login page with both options
	http.ServeFile(w, r, "templates/login.html")
}

func (h *AuthHandler) HandleLDAPLogin(w http.ResponseWriter, r *http.Request) {
	// Baca dan log isi body sekali
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read request body", http.StatusBadRequest)
		return
	}
	log.Println("📥 Raw Request Body:", string(bodyBytes))

	// Restore ulang body agar bisa dipakai untuk decode
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Decode ke struct
	var req LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Authenticate
	email, err := h.ldapClient.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Query employee ID
	var employeeID int
	err = h.db.Pool.QueryRow(ctx, `SELECT id FROM employees WHERE email = $1`, email).Scan(&employeeID)
	if err != nil {
		http.Error(w, "employee not found", http.StatusUnauthorized)
		return
	}

	// Fetch scopes
	rows, err := h.db.Pool.Query(ctx, `
		SELECT s.name FROM scopes s
		JOIN employee_scopes es ON es.scope_id = s.id
		WHERE es.employee_id = $1`, employeeID)
	if err != nil {
		http.Error(w, "failed to fetch scopes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scopeNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			http.Error(w, "error reading scopes", http.StatusInternalServerError)
			return
		}
		scopeNames = append(scopeNames, name)
	}

	// Generate token
	token, err := auth.GenerateToken(email, scopeNames, h.cfg)
	if err != nil {
		http.Error(w, "token generation error", http.StatusInternalServerError)
		return
	}

	// Set JWT token as HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "ldap_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(time.Hour.Seconds()), // 1 jam
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginRes{Token: token})
}

func (h *AuthHandler) HandleSSOLogin(w http.ResponseWriter, r *http.Request) {

	// 1. Check URL IdP
	idpURL := h.samlSP.ServiceProvider.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	if idpURL == "" {
		http.Error(w, "IdP SSO URL not configured", http.StatusInternalServerError)
		return
	}

	// 2.create AuthnRequest (target URL, binding request, binding response)
	authnRequest, err := h.samlSP.ServiceProvider.MakeAuthenticationRequest(
		idpURL,
		saml.HTTPRedirectBinding,
		saml.HTTPPostBinding,
	)
	if err != nil {
		h.samlSP.OnError(w, r, fmt.Errorf("make authn request: %w", err))
		return
	}

	// 3. Marshal AuthnRequest to XML
	xmlReq, err := xml.Marshal(authnRequest)
	if err != nil {
		h.samlSP.OnError(w, r, fmt.Errorf("marshal authn request xml: %w", err))
		return
	}

	// 4. DEFLATE + Base64 encode SAMLRequest
	var buf bytes.Buffer
	deflater, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		h.samlSP.OnError(w, r, fmt.Errorf("flate writer create: %w", err))
		return
	}
	if _, err := deflater.Write(xmlReq); err != nil {
		h.samlSP.OnError(w, r, fmt.Errorf("flate write: %w", err))
		return
	}
	deflater.Close()
	samlRequestEncoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// 5. RelayState
	relayState := "/"

	// 6. Buat URL redirect ke IdP dengan parameter
	redirectURL := fmt.Sprintf("%s?SAMLRequest=%s&RelayState=%s",
		idpURL,
		url.QueryEscape(samlRequestEncoded),
		url.QueryEscape(relayState),
	)

	// 7. Track relayState untuk validasi nanti
	if _, err := h.samlSP.RequestTracker.TrackRequest(w, r, relayState); err != nil {
		h.samlSP.OnError(w, r, fmt.Errorf("track request: %w", err))
		return
	}

	// 8. Redirect user ke IdP
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// 1️⃣ Hapus auth-session (biasa untuk SAML)
	authSession, _ := h.store.Get(r, "auth-session")
	authSession.Options.MaxAge = -1
	_ = authSession.Save(r, w)
	log.Println("✅ Cleared 'auth-session'")

	// 2️⃣ Hapus token session (jika ada)
	tokenSession, err := h.store.Get(r, "saml_token")
	if err == nil {
		tokenSession.Options.MaxAge = -1
		_ = tokenSession.Save(r, w)
		log.Println("✅ Cleared 'token' session")
	} else {
		// Fallback manual hapus cookie token
		http.SetCookie(w, &http.Cookie{
			Name:     "saml_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		log.Println("✅ Cleared 'token' cookie manually")
	}

	// 3️⃣ Hapus ldap_token cookie jika ada
	http.SetCookie(w, &http.Cookie{
		Name:     "ldap_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	log.Println("✅ Cleared 'ldap_token' cookie")

	// 4️⃣ Jika user login via SAML, redirect ke SAML logout
	if method, ok := authSession.Values["auth_method"].(string); ok && method == "saml" {
		log.Println("🔁 Redirecting to SAML logout")
		http.Redirect(w, r, "/saml/logout", http.StatusFound)
		return
	}

	// 5️⃣ Fallback redirect (LDAP logout or unknown)
	log.Println("🔁 Redirecting to /login")
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *AuthHandler) HybridAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) Try LDAP JWT
		cookie, err := r.Cookie("ldap_token")
		if err == nil && cookie.Value != "" {
			email, scopes, err := auth.ValidateToken(cookie.Value, h.cfg)
			if err == nil {
				// ✅ Token valid → inject context dan lanjut
				log.Printf("🔐 LDAP Authenticated: %s, scopes: %v\n", email, scopes)
				ctx := context.WithValue(r.Context(), "email", email)
				ctx = context.WithValue(ctx, "userScopes", scopes)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			} else {
				log.Printf("❌ Invalid JWT token: %v", err)
			}
		}

		// 2) If no valid LDAP, let SAML middleware handle
		samlCookie, samlErr := r.Cookie("saml_token")
		if samlErr == nil && samlCookie.Value != "" {
			//saml token
			h.samlSP.RequireAccount(next).ServeHTTP(w, r)
			return
		}

		session := samlsp.SessionFromContext(r.Context())
		if session != nil {
			log.Println("🔐 SAML session detected", session)
			next.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

func (h *AuthHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	// 🔍 Coba cek JWT (LDAP login)
	if email := r.Context().Value("email"); email != nil {
		scopes := r.Context().Value("userScopes")
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "✅ Logged in via LDAP JWT\n\n")
		fmt.Fprintf(w, "Email: %s\nScopes: %v\n", email, scopes)
		return
	}

	// 🔐 Cek SAML session
	session := samlsp.SessionFromContext(r.Context())
	if session == nil {
		// ❌ Tidak ada JWT, tidak ada SAML → redirect ke general login
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// 🧠 Pastikan session bertipe SAML
	samlSession, ok := session.(samlsp.SessionWithAttributes)
	if !ok {
		// ❌ Session ada tapi bukan session dengan atribut → redirect juga
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// ✅ SAML valid → tampilkan info
	attrs := samlSession.GetAttributes()
	uid := attrs.Get("uid")
	name := attrs.Get("displayName")
	email := attrs.Get("email")

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "✅ Logged in via SAML\n\n")
	fmt.Fprintf(w, "NameID: %s\nUID: %s\nEmail: %s\n\n", name, uid, email)

	// Print all attributes
	fmt.Fprintf(w, "All Attributes:\n")
	for key, attr := range attrs {
		fmt.Fprintf(w, "- %s: %v\n", key, attr)
	}
}
