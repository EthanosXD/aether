package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Simple signed token: base64(payload).base64(sig)
func signToken(userID int64) string {
	payload, _ := json.Marshal(map[string]interface{}{
		"id":  userID,
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := tokenSig(encoded)
	return encoded + "." + sig
}

func tokenSig(data string) string {
	mac := hmac.New(sha256.New, []byte(cfg.JWTSecret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseToken(token string) (int64, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token")
	}
	if tokenSig(parts[0]) != parts[1] {
		return 0, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, err
	}
	exp := int64(claims["exp"].(float64))
	if time.Now().Unix() > exp {
		return 0, fmt.Errorf("token expired")
	}
	return int64(claims["id"].(float64)), nil
}

func requireAuth(r *http.Request) (int64, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, fmt.Errorf("not logged in")
	}
	return parseToken(cookie.Value)
}

func setSession(w http.ResponseWriter, userID int64) {
	token := signToken(userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   30 * 24 * 3600,
	})
}

// ── Handlers ──

func handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || len(password) < 8 {
		respondJSON(w, 400, map[string]string{"error": "Email and password (min 8 chars) required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Server error"})
		return
	}

	id, err := createUser(email, string(hash))
	if err != nil {
		respondJSON(w, 409, map[string]string{"error": "Email already registered"})
		return
	}

	setSession(w, id)
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	id, hash, err := getUserByEmail(email)
	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		respondJSON(w, 401, map[string]string{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Server error"})
		return
	}

	setSession(w, id)
	respondJSON(w, 200, map[string]string{"status": "ok"})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	userID, err := requireAuth(r)
	if err != nil {
		respondJSON(w, 401, map[string]string{"error": "Unauthorized"})
		return
	}
	sub, err := getSubscription(userID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Server error"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{
		"user_id": userID,
		"tier":    sub.Tier,
		"status":  sub.Status,
		"license": sub.LicenseKey,
	})
}

// handleVerifyLicense is called by Aether nodes to check a license key
func handleVerifyLicense(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		respondJSON(w, 400, map[string]string{"error": "key required"})
		return
	}
	valid, tier, err := verifyLicenseKey(key)
	if err != nil {
		respondJSON(w, 404, map[string]string{"valid": "false"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{"valid": valid, "tier": tier})
}

func randomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
