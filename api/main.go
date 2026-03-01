package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const version = "0.1.0"

type Config struct {
	Port               string
	DBPath             string
	JWTSecret          string
	StripeSecretKey    string
	StripePriceID      string
	StripeWebhookSecret string
	BaseURL            string
}

var cfg Config

func main() {
	cfg = Config{
		Port:               getenv("PORT", "9090"),
		DBPath:             getenv("DB_PATH", "/tmp/aether.db"),
		JWTSecret:          getenv("JWT_SECRET", randomSecret()),
		StripeSecretKey:    getenv("STRIPE_SECRET_KEY", ""),
		StripePriceID:      getenv("STRIPE_PRICE_ID", ""),
		StripeWebhookSecret: getenv("STRIPE_WEBHOOK_SECRET", ""),
		BaseURL:            getenv("BASE_URL", "http://localhost:9090"),
	}

	if cfg.StripeSecretKey == "" {
		log.Println("WARNING: STRIPE_SECRET_KEY not set — payments disabled")
	}

	initDB(cfg.DBPath)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, map[string]string{"status": "ok", "version": version})
	})

	// Auth
	mux.HandleFunc("/api/signup",  handleSignup)
	mux.HandleFunc("/api/login",   handleLogin)
	mux.HandleFunc("/api/logout",  handleLogout)
	mux.HandleFunc("/api/me",      handleMe)

	// License verification (called by nodes)
	mux.HandleFunc("/api/license/verify", handleVerifyLicense)

	// Billing
	mux.HandleFunc("/api/checkout",       handleCheckout)
	mux.HandleFunc("/api/billing-portal", handleBillingPortal)
	mux.HandleFunc("/api/webhook",        handleStripeWebhook)

	// Frontend pages
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/login",     handleLoginPage)
	mux.HandleFunc("/signup",    handleSignupPage)
	mux.HandleFunc("/dashboard", handleDashboardPage)

	log.Printf("Aether API v%s on port %s", version, cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}

func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// handleIndex redirects to dashboard or login
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, err := requireAuth(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleLogin(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, loginHTML)
}

func handleSignupPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleSignup(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, signupHTML)
}

func handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	userID, err := requireAuth(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sub, err := getSubscription(userID)
	if err != nil {
		http.Error(w, "Server error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, renderDashboard(sub))
}
