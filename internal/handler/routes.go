package handler

import (
	"log"
	"net/http"
)

func SetupRoutes(h *AuthHandler) *http.Server {

	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	// Static file server with proper caching headers
	staticFS := http.FileServer(http.Dir("templates"))
	staticHandler := http.StripPrefix("/static/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			staticFS.ServeHTTP(w, r)
		}))

	mux := http.NewServeMux()

	mux.Handle("/saml/metadata", h.samlSP)
	mux.Handle("/saml/", loggingMiddleware(h.samlSP))

	mux.HandleFunc("/login", h.HandleLogin)
	mux.HandleFunc("/ldap-login", h.HandleLDAPLogin)
	mux.HandleFunc("/sso-login", h.HandleSSOLogin)

	mux.HandleFunc("/logout", h.HandleLogout)
	mux.HandleFunc("/", h.HybridAuthMiddleware(http.HandlerFunc(h.IndexHandler)).ServeHTTP)

	// Static files
	mux.Handle("/static/", loggingMiddleware(staticHandler))

	return &http.Server{
		Addr:    ":" + h.cfg.Port,
		Handler: mux,
	}
}
