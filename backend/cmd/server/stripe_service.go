package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

	"github.com/stripe/stripe-go/v79"
	billingportalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/subscription"
	"github.com/stripe/stripe-go/v79/webhook"
)

// StripeService handles all Stripe payment operations
type StripeService struct {
	userStore     UserStoreInterface
	webhookSecret string
	webhookStore  *WebhookEventStore
}

// NewStripeService creates a new Stripe service
func NewStripeService(userStore UserStoreInterface, webhookStore *WebhookEventStore) *StripeService {
	// Initialize Stripe with secret key
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	return &StripeService{
		userStore:     userStore,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		webhookStore:  webhookStore,
	}
}

// StripeProduct represents our pricing tiers as Stripe products
type StripeProduct struct {
	ProductID string
	PriceID   string
	Tier      string
	Name      string
	Price     int64 // Price in cents
}

// GetStripeProducts returns the Stripe product configuration for our tiers
func (ss *StripeService) GetStripeProducts() []StripeProduct {
	return []StripeProduct{
		{
			ProductID: os.Getenv("STRIPE_PLUS_PRODUCT_ID"),
			PriceID:   os.Getenv("STRIPE_PLUS_PRICE_ID"),
			Tier:      "plus",
			Name:      "Plus",
			Price:     999, // $9.99
		},
		{
			ProductID: os.Getenv("STRIPE_PRO_PRODUCT_ID"),
			PriceID:   os.Getenv("STRIPE_PRO_PRICE_ID"),
			Tier:      "pro",
			Name:      "Pro",
			Price:     1999, // $19.99
		},
		{
			ProductID: os.Getenv("STRIPE_PRO_PLUS_PRODUCT_ID"),
			PriceID:   os.Getenv("STRIPE_PRO_PLUS_PRICE_ID"),
			Tier:      "pro_plus",
			Name:      "Pro Plus",
			Price:     4999, // $49.99
		},
	}
}

// CreateCustomer creates a Stripe customer for a user
func (ss *StripeService) CreateCustomer(ctx context.Context, firebaseUID, email, displayName string) (*stripe.Customer, error) {
    log.Printf("[Stripe] Creating customer for user: %s", firebaseUID)

	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(displayName),
		Metadata: map[string]string{
			"firebase_uid": firebaseUID,
		},
	}

	customer, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("create stripe customer: %w", err)
	}

    log.Printf("[Stripe] Created customer: %s for user: %s", customer.ID, firebaseUID)
    return customer, nil
}

// CreateCheckoutSession creates a Stripe checkout session for subscription
func (ss *StripeService) CreateCheckoutSession(ctx context.Context, firebaseUID, tier, customerID string) (*stripe.CheckoutSession, error) {
	log.Printf("[Stripe] Creating checkout session for user: %s, tier: %s", firebaseUID, tier)

	// Find the price ID for the requested tier
	var priceID string
	for _, product := range ss.GetStripeProducts() {
		if product.Tier == tier {
			priceID = product.PriceID
			break
		}
	}

	if priceID == "" {
		return nil, fmt.Errorf("invalid tier: %s", tier)
	}

    // Require FRONTEND_URL for absolute redirect URLs
    frontendURL := os.Getenv("FRONTEND_URL")
    if strings.TrimSpace(frontendURL) == "" {
        return nil, fmt.Errorf("FRONTEND_URL is not configured")
    }

    // Create checkout session with explicit configuration
    params := &stripe.CheckoutSessionParams{
        Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
        Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
        SuccessURL: stripe.String(frontendURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}"),
        CancelURL:  stripe.String(frontendURL + "/billing/canceled"),
        Metadata: map[string]string{
            "firebase_uid": firebaseUID,
            "tier":         tier,
        },
    }

	// Note: Removed Locale, UIMode, and CustomerUpdate to use Stripe defaults
	// The './en' module error is a known Stripe test mode issue that doesn't affect functionality
	// The checkout page will still work correctly despite the console error

	session, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	log.Printf("[Stripe] Created checkout session: %s for user: %s", session.ID, firebaseUID)
	return session, nil
}

// CreateBillingPortalSession creates a session for customer billing portal
func (ss *StripeService) CreateBillingPortalSession(ctx context.Context, customerID string) (*stripe.BillingPortalSession, error) {
	log.Printf("[Stripe] Creating billing portal session for customer: %s", customerID)

    // Get frontend URL and require it in production
    frontendURL := os.Getenv("FRONTEND_URL")
    if strings.TrimSpace(frontendURL) == "" {
        return nil, fmt.Errorf("FRONTEND_URL is not configured")
    }
    returnURL := frontendURL + "/billing"
    log.Printf("[Stripe] Using return URL: %s", returnURL)

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := billingportalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("create billing portal session: %w", err)
	}

	log.Printf("[Stripe] Created billing portal session: %s", session.ID)
	return session, nil
}

// RenewCurrentSubscription manually renews the user's current subscription immediately
// This resets the billing period and token usage
func (ss *StripeService) RenewCurrentSubscription(ctx context.Context, firebaseUID string) error {
	log.Printf("[Stripe] Manual renewal requested for user: %s", firebaseUID)

	// Get user's current subscription
	user, err := ss.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if user.Subscription.StripeSubscriptionID == "" {
		return fmt.Errorf("user has no active subscription")
	}

	if user.Subscription.Tier == "free" {
		return fmt.Errorf("cannot renew free tier subscription")
	}

	// Get the current Stripe subscription
	sub, err := subscription.Get(user.Subscription.StripeSubscriptionID, nil)
	if err != nil {
		return fmt.Errorf("get stripe subscription: %w", err)
	}

	log.Printf("[Stripe] Current subscription status: %s, period: %d - %d",
		sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

	// Update subscription to start new billing period immediately
	// Stripe requires the string "now" instead of a timestamp for updates
	params := &stripe.SubscriptionParams{
		BillingCycleAnchorNow: stripe.Bool(true),
		ProrationBehavior:     stripe.String("always_invoice"),
	}

	updatedSub, err := subscription.Update(sub.ID, params)
	if err != nil {
		return fmt.Errorf("update subscription billing cycle: %w", err)
	}

	log.Printf("[Stripe] Updated subscription: new period %d - %d",
		updatedSub.CurrentPeriodStart, updatedSub.CurrentPeriodEnd)

	// Reset token usage in database
	err = ss.userStore.ResetUsageForBillingPeriod(ctx, firebaseUID)
	if err != nil {
		log.Printf("[Stripe] Warning: Failed to reset token usage: %v", err)
	}

	// Update subscription details in database
	newSub := &Subscription{
		Tier:                 user.Subscription.Tier,
		Status:               string(updatedSub.Status),
		StripeCustomerID:     user.Subscription.StripeCustomerID,
		StripeSubscriptionID: updatedSub.ID,
		CurrentPeriodStart:   updatedSub.CurrentPeriodStart,
		CurrentPeriodEnd:     updatedSub.CurrentPeriodEnd,
		CancelAtPeriodEnd:    updatedSub.CancelAtPeriodEnd,
	}

	err = ss.userStore.UpdateSubscription(ctx, firebaseUID, newSub)
	if err != nil {
		return fmt.Errorf("update subscription in database: %w", err)
	}

	log.Printf("[Stripe] ✅ Successfully renewed subscription for user %s - new period starts now", firebaseUID)
	return nil
}

// HandleWebhook processes Stripe webhook events
func (ss *StripeService) HandleWebhook(payload []byte, signature string) error {
	log.Printf("[Stripe] 🔄 Starting webhook processing...")
	log.Printf("[Stripe] Payload size: %d bytes", len(payload))
	log.Printf("[Stripe] Signature: %s", signature)
	log.Printf("[Stripe] Webhook secret configured: %t", ss.webhookSecret != "")

	log.Printf("[Stripe] 🔐 Verifying webhook signature...")
	event, err := webhook.ConstructEventWithOptions(payload, signature, ss.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("[Stripe] ❌ Webhook signature verification FAILED: %v", err)
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	log.Printf("[Stripe] ✅ Webhook signature verification SUCCESS")
	log.Printf("[Stripe] Event ID: %s", event.ID)
	log.Printf("[Stripe] Event type: %s", event.Type)
	log.Printf("[Stripe] Event created: %d", event.Created)
	log.Printf("[Stripe] Event API version: %s", event.APIVersion)

	// ✅ CHECK FOR DUPLICATE EVENT (Idempotency)
	ctx := context.Background()
	processed, err := ss.webhookStore.IsEventProcessed(ctx, event.ID)
	if err != nil {
		log.Printf("[Stripe] ⚠️ Error checking event status: %v", err)
		// Continue processing - better to risk duplicate than lose event
	} else if processed {
		log.Printf("[Stripe] ⏭️ Event %s already processed, skipping to prevent duplicate charge", event.ID)
		return nil // Return success without processing
	}

	log.Printf("[Stripe] 🎯 Routing event type: %s", event.Type)

	// Process the event and capture any error
	var processingErr error
	switch event.Type {
	case "checkout.session.completed":
		log.Printf("[Stripe] 🛒 Processing checkout.session.completed event")
		processingErr = ss.handleCheckoutCompleted(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ checkout.session.completed handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ checkout.session.completed handler succeeded")
		}
	case "customer.subscription.created":
		log.Printf("[Stripe] 📝 Processing customer.subscription.created event")
		processingErr = ss.handleSubscriptionCreated(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ subscription.created handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ subscription.created handler succeeded")
		}
	case "customer.subscription.updated":
		log.Printf("[Stripe] 🔄 Processing customer.subscription.updated event")
		processingErr = ss.handleSubscriptionUpdated(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ subscription.updated handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ subscription.updated handler succeeded")
		}
	case "customer.subscription.deleted":
		log.Printf("[Stripe] 🗑️ Processing customer.subscription.deleted event")
		processingErr = ss.handleSubscriptionDeleted(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ subscription.deleted handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ subscription.deleted handler succeeded")
		}
	case "invoice.payment_succeeded":
		log.Printf("[Stripe] 💰 Processing invoice.payment_succeeded event")
		processingErr = ss.handlePaymentSucceeded(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ payment.succeeded handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ payment.succeeded handler succeeded")
		}
	case "invoice.payment_failed":
		log.Printf("[Stripe] 💳 Processing invoice.payment_failed event")
		processingErr = ss.handlePaymentFailed(event)
		if processingErr != nil {
			log.Printf("[Stripe] ❌ payment.failed handler failed: %v", processingErr)
		} else {
			log.Printf("[Stripe] ✅ payment.failed handler succeeded")
		}
	default:
		log.Printf("[Stripe] ⚠️ Unhandled webhook event type: %s (ignoring)", event.Type)
	}

	// ✅ MARK EVENT AS PROCESSED (only if processing succeeded)
	if processingErr == nil {
		if err := ss.webhookStore.MarkEventProcessed(ctx, event.ID, string(event.Type)); err != nil {
			log.Printf("[Stripe] ⚠️ Failed to mark event %s as processed: %v", event.ID, err)
			// Don't fail the webhook - event was already processed successfully
			// Worst case: event gets processed again if Stripe retries
		} else {
			log.Printf("[Stripe] ✅ Event %s marked as processed in database", event.ID)
		}
	} else {
		log.Printf("[Stripe] ⚠️ Event %s NOT marked as processed due to processing error", event.ID)
	}

	log.Printf("[Stripe] ✅ Webhook processing completed successfully")
	return processingErr
}

// handleSubscriptionCreated processes subscription creation webhook
func (ss *StripeService) handleSubscriptionCreated(event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("parse subscription created event: %w", err)
	}

	log.Printf("[Stripe] Subscription created: %s for customer: %s", subscription.ID, subscription.Customer.ID)

	// Get Firebase UID from customer metadata
	customer, err := customer.Get(subscription.Customer.ID, nil)
	if err != nil {
		return fmt.Errorf("get customer for subscription: %w", err)
	}

	firebaseUID := customer.Metadata["firebase_uid"]
	if firebaseUID == "" {
		return fmt.Errorf("firebase_uid not found in customer metadata")
	}

	// Determine tier from subscription
	tier := ss.getTierFromSubscription(&subscription)
	if tier == "" {
		return fmt.Errorf("could not determine tier from subscription")
	}

	// Update user subscription
	now := time.Now().UnixMilli()
	userSubscription := &Subscription{
		Tier:                 tier,
		Status:               string(subscription.Status),
		StripeCustomerID:     subscription.Customer.ID,
		StripeSubscriptionID: subscription.ID,
		CurrentPeriodStart:   subscription.CurrentPeriodStart * 1000, // Convert to milliseconds
		CurrentPeriodEnd:     subscription.CurrentPeriodEnd * 1000,   // Convert to milliseconds
		CancelAtPeriodEnd:    subscription.CancelAtPeriodEnd,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = ss.userStore.UpdateSubscription(ctx, firebaseUID, userSubscription)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	// Reset usage for the new billing period - CRITICAL: This must succeed
	log.Printf("[Stripe] 🔄 Resetting usage for new billing period (subscription created)...")
	err = ss.userStore.ResetUsageForBillingPeriod(ctx, firebaseUID)
	if err != nil {
		log.Printf("[Stripe] ❌ CRITICAL: Failed to reset usage for user %s: %v", firebaseUID, err)
		return fmt.Errorf("reset usage for new billing period: %w", err)
	} else {
		log.Printf("[Stripe] ✅ Usage reset succeeded (subscription created)")
	}

	return nil
}

// handleSubscriptionUpdated processes subscription update webhook
func (ss *StripeService) handleSubscriptionUpdated(event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("parse subscription updated event: %w", err)
	}

	log.Printf("[Stripe] Subscription updated: %s", subscription.ID)

	// Get Firebase UID from customer metadata
	customer, err := customer.Get(subscription.Customer.ID, nil)
	if err != nil {
		return fmt.Errorf("get customer for subscription update: %w", err)
	}

	firebaseUID := customer.Metadata["firebase_uid"]
	if firebaseUID == "" {
		return fmt.Errorf("firebase_uid not found in customer metadata")
	}

	// Determine tier from subscription
	tier := ss.getTierFromSubscription(&subscription)
	if tier == "" {
		return fmt.Errorf("could not determine tier from subscription")
	}

	// Update user subscription
	now := time.Now().UnixMilli()
	userSubscription := &Subscription{
		Tier:                 tier,
		Status:               string(subscription.Status),
		StripeCustomerID:     subscription.Customer.ID,
		StripeSubscriptionID: subscription.ID,
		CurrentPeriodStart:   subscription.CurrentPeriodStart * 1000,
		CurrentPeriodEnd:     subscription.CurrentPeriodEnd * 1000,
		CancelAtPeriodEnd:    subscription.CancelAtPeriodEnd,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = ss.userStore.UpdateSubscription(ctx, firebaseUID, userSubscription)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	// Reset usage for the new billing period - CRITICAL: This must succeed
	log.Printf("[Stripe] 🔄 Resetting usage for new billing period (subscription updated)...")
	err = ss.userStore.ResetUsageForBillingPeriod(ctx, firebaseUID)
	if err != nil {
		log.Printf("[Stripe] ❌ CRITICAL: Failed to reset usage for user %s: %v", firebaseUID, err)
		return fmt.Errorf("reset usage for new billing period: %w", err)
	} else {
		log.Printf("[Stripe] ✅ Usage reset succeeded (subscription updated)")
	}

	return nil
}

// handleSubscriptionDeleted processes subscription cancellation webhook
func (ss *StripeService) handleSubscriptionDeleted(event stripe.Event) error {
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return fmt.Errorf("parse subscription deleted event: %w", err)
	}

	log.Printf("[Stripe] Subscription deleted: %s", subscription.ID)

	// Get Firebase UID from customer metadata
	customer, err := customer.Get(subscription.Customer.ID, nil)
	if err != nil {
		return fmt.Errorf("get customer for subscription deletion: %w", err)
	}

	firebaseUID := customer.Metadata["firebase_uid"]
	if firebaseUID == "" {
		return fmt.Errorf("firebase_uid not found in customer metadata")
	}

	// Downgrade to free tier
	userSubscription := &Subscription{
		Tier:                 "free",
		Status:               "canceled",
		StripeCustomerID:     subscription.Customer.ID,
		StripeSubscriptionID: "",
		CurrentPeriodStart:   time.Now().UnixMilli(),
		CurrentPeriodEnd:     time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
		CancelAtPeriodEnd:    false,
		UpdatedAt:            time.Now().UnixMilli(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return ss.userStore.UpdateSubscription(ctx, firebaseUID, userSubscription)
}

// handlePaymentSucceeded processes successful payment webhook
func (ss *StripeService) handlePaymentSucceeded(event stripe.Event) error {
	log.Printf("[Stripe] Payment succeeded for event: %s", event.ID)
	// Could implement additional logic here (notifications, analytics, etc.)
	return nil
}

// handlePaymentFailed processes failed payment webhook
func (ss *StripeService) handlePaymentFailed(event stripe.Event) error {
	log.Printf("[Stripe] Payment failed for event: %s", event.ID)
	// Could implement additional logic here (notifications, retry logic, etc.)
	return nil
}

// handleCheckoutCompleted processes completed checkout session webhook
func (ss *StripeService) handleCheckoutCompleted(event stripe.Event) error {
	log.Printf("[Stripe] 🛒 Starting handleCheckoutCompleted...")

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("[Stripe] ❌ Failed to parse checkout session event: %v", err)
		return fmt.Errorf("parse checkout session completed event: %w", err)
	}

	log.Printf("[Stripe] ✅ Parsed checkout session successfully")
	log.Printf("[Stripe] Session ID: %s", session.ID)
	log.Printf("[Stripe] Customer ID: %s", session.Customer.ID)
	log.Printf("[Stripe] Payment status: %s", session.PaymentStatus)
	log.Printf("[Stripe] Session mode: %s", session.Mode)
	log.Printf("[Stripe] Session status: %s", session.Status)

	// Get customer details to find Firebase UID
	log.Printf("[Stripe] 👤 Fetching customer details for ID: %s", session.Customer.ID)
	customer, err := customer.Get(session.Customer.ID, nil)
	if err != nil {
		log.Printf("[Stripe] ❌ Failed to get customer: %v", err)
		return fmt.Errorf("get customer for checkout session: %w", err)
	}

    log.Printf("[Stripe] ✅ Retrieved customer: %s", customer.ID)
    // Avoid logging customer email or full metadata in production logs

	firebaseUID := customer.Metadata["firebase_uid"]
	if firebaseUID == "" {
		log.Printf("[Stripe] ❌ firebase_uid not found in customer metadata")
		var keys []string
		for k := range customer.Metadata {
			keys = append(keys, k)
		}
		log.Printf("[Stripe] Available metadata keys: %v", keys)
		return fmt.Errorf("firebase_uid not found in customer metadata")
	}

	log.Printf("[Stripe] ✅ Found Firebase UID: %s", firebaseUID)

	// Get subscription from the session
	if session.Subscription == nil {
		log.Printf("[Stripe] ⚠️ No subscription in checkout session: %s (this is normal for one-time payments)", session.ID)
		return nil
	}

	log.Printf("[Stripe] 📋 Found subscription in session: %s", session.Subscription.ID)

	// Retrieve the subscription details
	log.Printf("[Stripe] 🔍 Fetching subscription details...")
	subscription, err := subscription.Get(session.Subscription.ID, nil)
	if err != nil {
		log.Printf("[Stripe] ❌ Failed to get subscription: %v", err)
		return fmt.Errorf("get subscription from checkout session: %w", err)
	}

	log.Printf("[Stripe] ✅ Retrieved subscription: %s", subscription.ID)
	log.Printf("[Stripe] Subscription status: %s", subscription.Status)
	log.Printf("[Stripe] Subscription items count: %d", len(subscription.Items.Data))

	// Determine tier from subscription
	log.Printf("[Stripe] 🎯 Determining tier from subscription...")
	tier := ss.getTierFromSubscription(subscription)
	if tier == "" {
		log.Printf("[Stripe] ❌ Could not determine tier from subscription: %s", subscription.ID)
		if len(subscription.Items.Data) > 0 {
			log.Printf("[Stripe] Price ID: %s", subscription.Items.Data[0].Price.ID)
		}
		return fmt.Errorf("unknown price ID for subscription: %s", subscription.ID)
	}

    log.Printf("[Stripe] ✅ Determined tier: %s", tier)
    log.Printf("[Stripe] 💾 Updating user subscription for user: %s", firebaseUID)

	// Update user subscription in database
	subscriptionData := &Subscription{
		Tier:                 tier,
		Status:               string(subscription.Status),
		StripeCustomerID:     customer.ID,
		StripeSubscriptionID: subscription.ID,
		CurrentPeriodStart:   subscription.CurrentPeriodStart * 1000, // Convert to milliseconds
		CurrentPeriodEnd:     subscription.CurrentPeriodEnd * 1000,   // Convert to milliseconds
		CancelAtPeriodEnd:    subscription.CancelAtPeriodEnd,
		CreatedAt:            time.Now().UnixMilli(),
		UpdatedAt:            time.Now().UnixMilli(),
	}

	log.Printf("[Stripe] 📝 Subscription data to save: Tier=%s, Status=%s, Customer=%s, Subscription=%s",
		subscriptionData.Tier, subscriptionData.Status, subscriptionData.StripeCustomerID, subscriptionData.StripeSubscriptionID)
	log.Printf("[Stripe] Period: %d - %d", subscriptionData.CurrentPeriodStart, subscriptionData.CurrentPeriodEnd)

	// Update subscription using the userStore interface
	log.Printf("[Stripe] 🔄 Calling userStore.UpdateSubscription...")
	err = ss.userStore.UpdateSubscription(context.Background(), firebaseUID, subscriptionData)
	if err != nil {
		log.Printf("[Stripe] ❌ UpdateSubscription failed: %v", err)
		return fmt.Errorf("update subscription: %w", err)
	}
	log.Printf("[Stripe] ✅ UpdateSubscription succeeded")

	// Reset usage for the new billing period - CRITICAL: This must succeed
	log.Printf("[Stripe] 🔄 Resetting usage for new billing period...")
	err = ss.userStore.ResetUsageForBillingPeriod(context.Background(), firebaseUID)
	if err != nil {
		log.Printf("[Stripe] ❌ CRITICAL: Failed to reset usage for user %s: %v", firebaseUID, err)
		return fmt.Errorf("reset usage for new billing period: %w", err)
	} else {
		log.Printf("[Stripe] ✅ Usage reset succeeded")
	}

	log.Printf("[Stripe] 🎉 Successfully updated subscription for user %s to tier: %s", firebaseUID, tier)
	return nil
}

// getTierFromSubscription determines the tier from a Stripe subscription
func (ss *StripeService) getTierFromSubscription(subscription *stripe.Subscription) string {
	log.Printf("[Stripe] 🔍 getTierFromSubscription: subscription %s", subscription.ID)

	if len(subscription.Items.Data) == 0 {
		log.Printf("[Stripe] ❌ No subscription items found")
		return ""
	}

	priceID := subscription.Items.Data[0].Price.ID
	log.Printf("[Stripe] 💰 Price ID to match: %s", priceID)

	// Map price ID to tier
	products := ss.GetStripeProducts()
	log.Printf("[Stripe] 📋 Available products: %d", len(products))

	for i, product := range products {
		log.Printf("[Stripe] Product %d: PriceID=%s, Tier=%s", i, product.PriceID, product.Tier)
		if product.PriceID == priceID {
			log.Printf("[Stripe] ✅ Found matching tier: %s for price ID: %s", product.Tier, priceID)
			return product.Tier
		}
	}

	log.Printf("[Stripe] ❌ No matching tier found for price ID: %s", priceID)
	return ""
}

// GetUserSubscriptionStatus retrieves subscription status for a user
func (ss *StripeService) GetUserSubscriptionStatus(ctx context.Context, firebaseUID string) (*SubscriptionStatus, error) {
	log.Printf("[Stripe] GetUserSubscriptionStatus called for user: %s", firebaseUID)

	user, err := ss.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		log.Printf("[Stripe] Failed to get user %s: %v", firebaseUID, err)
		return nil, fmt.Errorf("get user for subscription status: %w", err)
	}

	log.Printf("[Stripe] User %s retrieved successfully", firebaseUID)
	log.Printf("[Stripe] User subscription tier: '%s'", user.Subscription.Tier)
	log.Printf("[Stripe] User subscription status: '%s'", user.Subscription.Status)
	log.Printf("[Stripe] User subscription Stripe ID: '%s'", user.Subscription.StripeSubscriptionID)

	now := time.Now()
	freeTrialActive := user.FreeTrialActive(now)

	// Handle case where user has no subscription yet (Tier is empty for uninitialized subscriptions)
	if user.Subscription.Tier == "" {
		log.Printf("[Stripe] User %s has no subscription tier, returning free tier", firebaseUID)
		effectiveTier := user.EffectiveTier(now)
		return &SubscriptionStatus{
			Tier:              effectiveTier,
			Status:            "inactive",
			CurrentPeriodEnd:  0,
			CancelAtPeriodEnd: false,
			HasPaymentMethod:  false,
			FreeTrialActive:   freeTrialActive,
			FreeTrialStartAt:  user.FreeTrial.StartAt,
			FreeTrialEndAt:    user.FreeTrial.EndAt,
		}, nil
	}

    displayTier := user.EffectiveTier(now)
    if freeTrialActive && user.Subscription.StripeSubscriptionID == "" {
        displayTier = "free_trial"
    }

    status := &SubscriptionStatus{
        Tier:              displayTier,
        Status:            user.Subscription.Status,
        CurrentPeriodEnd:  user.Subscription.CurrentPeriodEnd,
        CancelAtPeriodEnd: user.Subscription.CancelAtPeriodEnd,
        HasPaymentMethod:  user.Subscription.StripeCustomerID != "",
        FreeTrialActive:   freeTrialActive,
        FreeTrialStartAt:  user.FreeTrial.StartAt,
        FreeTrialEndAt:    user.FreeTrial.EndAt,
    }

    if status.FreeTrialActive && user.Subscription.StripeSubscriptionID == "" {
        if status.FreeTrialEndAt > 0 {
            status.CurrentPeriodEnd = status.FreeTrialEndAt
            status.NextBillingDate = status.FreeTrialEndAt
        }
    }

	// If user has a Stripe subscription, get additional details
	if user.Subscription.StripeSubscriptionID != "" {
		log.Printf("[Stripe] Getting additional details for Stripe subscription: %s", user.Subscription.StripeSubscriptionID)
		stripeSubscription, err := subscription.Get(user.Subscription.StripeSubscriptionID, nil)
		if err != nil {
			log.Printf("[Stripe] Failed to get subscription details: %v", err)
		} else {
			log.Printf("[Stripe] Live Stripe subscription status: %s", stripeSubscription.Status)
			log.Printf("[Stripe] Database subscription status: %s", user.Subscription.Status)
			log.Printf("[Stripe] Live Stripe CurrentPeriodEnd: %d", stripeSubscription.CurrentPeriodEnd)
			log.Printf("[Stripe] Live Stripe CancelAtPeriodEnd: %t", stripeSubscription.CancelAtPeriodEnd)

			// Use live Stripe status and dates instead of database values
			status.Status = string(stripeSubscription.Status)
			status.CurrentPeriodEnd = stripeSubscription.CurrentPeriodEnd * 1000 // Convert to milliseconds
			status.NextBillingDate = stripeSubscription.CurrentPeriodEnd * 1000
			status.PastDue = stripeSubscription.Status == stripe.SubscriptionStatusPastDue
			status.CancelAtPeriodEnd = stripeSubscription.CancelAtPeriodEnd

			// If Stripe status is different from database, log this discrepancy
			if string(stripeSubscription.Status) != user.Subscription.Status {
				log.Printf("[Stripe] ⚠️  STATUS MISMATCH - Stripe: %s, Database: %s", stripeSubscription.Status, user.Subscription.Status)
				log.Printf("[Stripe] Using live Stripe status: %s", stripeSubscription.Status)
			}
		}
	}

	return status, nil
}

// CancelSubscription cancels a user's subscription at the end of the current period
func (ss *StripeService) CancelSubscription(ctx context.Context, firebaseUID string) error {
	log.Printf("[Stripe] CancelSubscription called for user: %s", firebaseUID)

	user, err := ss.userStore.GetUser(ctx, firebaseUID)
	if err != nil {
		log.Printf("[Stripe] Failed to get user %s: %v", firebaseUID, err)
		return fmt.Errorf("get user for subscription cancellation: %w", err)
	}

	if user.Subscription.StripeSubscriptionID == "" {
		log.Printf("[Stripe] User %s has no Stripe subscription to cancel", firebaseUID)
		return fmt.Errorf("no active subscription found")
	}

	log.Printf("[Stripe] Canceling subscription: %s", user.Subscription.StripeSubscriptionID)

	// Cancel the subscription at period end (not immediately)
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	subscription, err := subscription.Update(user.Subscription.StripeSubscriptionID, params)
	if err != nil {
		log.Printf("[Stripe] Failed to cancel subscription %s: %v", user.Subscription.StripeSubscriptionID, err)
		return fmt.Errorf("cancel subscription in Stripe: %w", err)
	}

	log.Printf("[Stripe] ✅ Subscription %s marked for cancellation at period end", subscription.ID)
	log.Printf("[Stripe] Subscription will end on: %d", subscription.CurrentPeriodEnd)

	// Update user subscription in database
	now := time.Now().UnixMilli()
	userSubscription := &Subscription{
		Tier:                 user.Subscription.Tier,      // Keep current tier until period ends
		Status:               string(subscription.Status), // Should still be 'active'
		StripeCustomerID:     user.Subscription.StripeCustomerID,
		StripeSubscriptionID: user.Subscription.StripeSubscriptionID,
		CurrentPeriodStart:   user.Subscription.CurrentPeriodStart,
		CurrentPeriodEnd:     user.Subscription.CurrentPeriodEnd,
		CancelAtPeriodEnd:    subscription.CancelAtPeriodEnd, // This will be true now
		CreatedAt:            user.Subscription.CreatedAt,
		UpdatedAt:            now,
	}

	err = ss.userStore.UpdateSubscription(ctx, firebaseUID, userSubscription)
	if err != nil {
		log.Printf("[Stripe] Failed to update user subscription in database: %v", err)
		return fmt.Errorf("update user subscription: %w", err)
	}

	log.Printf("[Stripe] ✅ User subscription updated - will cancel at period end")
	return nil
}

// SubscriptionStatus contains subscription information for frontend
type SubscriptionStatus struct {
	Tier              string `json:"tier"`
	Status            string `json:"status"`
	CurrentPeriodEnd  int64  `json:"currentPeriodEnd"`
	NextBillingDate   int64  `json:"nextBillingDate"`
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`
	HasPaymentMethod  bool   `json:"hasPaymentMethod"`
	PastDue           bool   `json:"pastDue"`
	FreeTrialActive   bool   `json:"freeTrialActive"`
	FreeTrialStartAt  int64  `json:"freeTrialStartAt,omitempty"`
	FreeTrialEndAt    int64  `json:"freeTrialEndAt,omitempty"`
}
