package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireIdentityWithAuthenticatedUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	authCtx := &AuthContext{UserID: "user-123"}
	req = req.WithContext(context.WithValue(req.Context(), AuthContextKey, authCtx))

	rr := httptest.NewRecorder()

	identity, ok := requireIdentity(rr, req)
	if !ok {
		t.Fatalf("expected identity to be resolved")
	}

	if identity == nil {
		t.Fatalf("expected non-nil identity")
	}

	if identity.UserID != "user-123" {
		t.Fatalf("expected userID 'user-123', got %q", identity.UserID)
	}

	if identity.Kind != IdentityKindAuthenticated {
		t.Fatalf("expected identity kind %q, got %q", IdentityKindAuthenticated, identity.Kind)
	}

	if rr.Code != http.StatusOK && rr.Code != 0 {
		t.Fatalf("unexpected status code written: %d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", rr.Body.String())
	}
}

func TestRequireIdentityWithoutCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	rr := httptest.NewRecorder()

	if identity, ok := requireIdentity(rr, req); ok || identity != nil {
		t.Fatalf("expected identity resolution to fail, got: %#v", identity)
	}

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if body := rr.Body.String(); body != "authentication required\n" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestRequireIdentityWithAnonymousHeaderButNoStore(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	req.Header.Set(identityHeaderAnonymous, "anon-user-1")
	rr := httptest.NewRecorder()

	if identity, ok := requireIdentity(rr, req); ok || identity != nil {
		t.Fatalf("expected identity resolution to fail for missing store")
	}

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	if body := rr.Body.String(); body != "invalid or expired session\n" {
		t.Fatalf("unexpected body %q", body)
	}
}
