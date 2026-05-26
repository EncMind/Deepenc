package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	identityHeaderAnonymous = "X-Anonymous-User-ID"
)

// IdentityKind describes the type of caller identity attached to the request.
type IdentityKind string

const (
	IdentityKindAuthenticated IdentityKind = "authenticated"
	IdentityKindAnonymous     IdentityKind = "anonymous"
)

// RequestIdentity captures the resolved identity for the current request.
type RequestIdentity struct {
	UserID string
	Kind   IdentityKind
}

var errIdentityRequired = errors.New("identity is required")

// resolveRequestIdentity determines the caller identity from the request context.
// Priority:
//  1. Authenticated Firebase user (Authorization header processed by AuthMiddleware)
//  2. Anonymous session ID supplied via X-Anonymous-User-ID header (validated against storage)
func resolveRequestIdentity(r *http.Request) (*RequestIdentity, error) {
	if authCtx := GetAuthContext(r.Context()); authCtx != nil && authCtx.UserID != "" {
		return &RequestIdentity{
			UserID: authCtx.UserID,
			Kind:   IdentityKindAuthenticated,
		}, nil
	}

	anonID := strings.TrimSpace(r.Header.Get(identityHeaderAnonymous))
	if anonID == "" {
		return nil, errIdentityRequired
	}

	if anonymousUserStore == nil {
		return nil, fmt.Errorf("anonymous sessions are not enabled")
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Validate the anonymous session exists; we do not auto-create here to avoid spoofing.
	if _, err := anonymousUserStore.GetAnonymousUser(ctx, anonID); err != nil {
		return nil, fmt.Errorf("invalid anonymous session: %w", err)
	}

	return &RequestIdentity{
		UserID: anonID,
		Kind:   IdentityKindAnonymous,
	}, nil
}

// requireIdentity resolves the caller identity and writes the appropriate HTTP response when missing.
func requireIdentity(w http.ResponseWriter, r *http.Request) (*RequestIdentity, bool) {
	identity, err := resolveRequestIdentity(r)
	if err == nil {
		return identity, true
	}

	status := http.StatusUnauthorized
	msg := "authentication required"
	if !errors.Is(err, errIdentityRequired) {
		status = http.StatusForbidden
		msg = "invalid or expired session"
	}

	log.Printf("[Identity] Request identity validation failed: %v (status=%d)", err, status)
	http.Error(w, msg, status)
	return nil, false
}
