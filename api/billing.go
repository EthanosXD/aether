package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	stripe "github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	userID, err := requireAuth(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stripe.Key = cfg.StripeSecretKey

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(cfg.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(cfg.BaseURL + "/dashboard?success=1"),
		CancelURL:  stripe.String(cfg.BaseURL + "/dashboard?cancelled=1"),
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", userID),
		},
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		log.Printf("Stripe checkout error: %v", err)
		respondJSON(w, 500, map[string]string{"error": "Payment setup failed"})
		return
	}

	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

func handleBillingPortal(w http.ResponseWriter, r *http.Request) {
	userID, err := requireAuth(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	sub, err := getSubscription(userID)
	if err != nil || sub.Tier != "pro" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	stripe.Key = cfg.StripeSecretKey
	params := &stripe.BillingPortalSessionParams{
		ReturnURL: stripe.String(cfg.BaseURL + "/dashboard"),
	}

	// We need the customer ID - stored in the DB
	var customerID string
	db.QueryRow(`SELECT stripe_customer_id FROM subscriptions WHERE user_id=?`, userID).Scan(&customerID)
	if customerID == "" {
		respondJSON(w, 400, map[string]string{"error": "No billing account found"})
		return
	}
	params.Customer = stripe.String(customerID)

	s, err := portalsession.New(params)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Portal failed"})
		return
	}
	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), cfg.StripeWebhookSecret)
	if err != nil {
		log.Printf("Webhook signature error: %v", err)
		w.WriteHeader(400)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var s stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &s); err != nil {
			break
		}
		userIDStr := s.Metadata["user_id"]
		var userID int64
		fmt.Sscanf(userIDStr, "%d", &userID)

		expires := time.Unix(s.Subscription.CurrentPeriodEnd, 0)
		key, err := activateProSubscription(userID, s.Customer.ID, s.Subscription.ID, expires)
		if err != nil {
			log.Printf("Activate subscription error: %v", err)
		} else {
			log.Printf("Pro activated for user %d — key: %s", userID, key)
		}

	case "customer.subscription.deleted", "customer.subscription.paused":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			break
		}
		if err := deactivateSubscription(sub.ID); err != nil {
			log.Printf("Deactivate error: %v", err)
		} else {
			log.Printf("Subscription deactivated: %s", sub.ID)
		}
	}

	w.WriteHeader(200)
}
