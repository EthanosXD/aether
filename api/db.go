package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) {
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("DB open error: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			email       TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS subscriptions (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id              INTEGER UNIQUE REFERENCES users(id),
			stripe_customer_id   TEXT,
			stripe_sub_id        TEXT,
			license_key          TEXT UNIQUE,
			tier                 TEXT DEFAULT 'free',
			status               TEXT DEFAULT 'inactive',
			expires_at           DATETIME,
			created_at           DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatalf("DB schema error: %v", err)
	}
	log.Println("Database ready")
}

// ── Users ──

func createUser(email, passwordHash string) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO users (email, password) VALUES (?, ?)`,
		email, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	// Create a free subscription record for new users
	_, err = db.Exec(
		`INSERT INTO subscriptions (user_id, tier, status) VALUES (?, 'free', 'active')`,
		id,
	)
	return id, err
}

func getUserByEmail(email string) (id int64, passwordHash string, err error) {
	err = db.QueryRow(
		`SELECT id, password FROM users WHERE email = ?`, email,
	).Scan(&id, &passwordHash)
	return
}

// ── Subscriptions ──

type Subscription struct {
	Tier      string
	Status    string
	LicenseKey string
	ExpiresAt *time.Time
}

func getSubscription(userID int64) (*Subscription, error) {
	sub := &Subscription{}
	var key sql.NullString
	var expires sql.NullTime
	err := db.QueryRow(
		`SELECT tier, status, license_key, expires_at FROM subscriptions WHERE user_id = ?`,
		userID,
	).Scan(&sub.Tier, &sub.Status, &key, &expires)
	if err != nil {
		return nil, err
	}
	if key.Valid {
		sub.LicenseKey = key.String
	}
	if expires.Valid {
		sub.ExpiresAt = &expires.Time
	}
	return sub, nil
}

func activateProSubscription(userID int64, stripeCustomerID, stripeSubID string, expiresAt time.Time) (string, error) {
	key := generateLicenseKey()
	_, err := db.Exec(`
		UPDATE subscriptions
		SET tier='pro', status='active', license_key=?, stripe_customer_id=?, stripe_sub_id=?, expires_at=?
		WHERE user_id=?
	`, key, stripeCustomerID, stripeSubID, expiresAt, userID)
	return key, err
}

func deactivateSubscription(stripeSubID string) error {
	_, err := db.Exec(`
		UPDATE subscriptions SET status='inactive', tier='free' WHERE stripe_sub_id=?
	`, stripeSubID)
	return err
}

func verifyLicenseKey(key string) (bool, string, error) {
	var tier, status string
	var expires sql.NullTime
	err := db.QueryRow(`
		SELECT tier, status, expires_at FROM subscriptions WHERE license_key=?
	`, key).Scan(&tier, &status, &expires)
	if err != nil {
		return false, "", err
	}
	if status != "active" {
		return false, tier, nil
	}
	if expires.Valid && time.Now().After(expires.Time) {
		return false, tier, nil
	}
	return true, tier, nil
}

func generateLicenseKey() string {
	b := make([]byte, 16)
	rand.Read(b)
	raw := hex.EncodeToString(b)
	// Format as AETH-XXXX-XXXX-XXXX-XXXX
	return "AETH-" + raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12] + "-" + raw[12:16]
}
