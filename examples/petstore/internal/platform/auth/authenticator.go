package auth

import (
	"context"
	"net/http"
	"strings"

	apierrors "github.com/fromforgesoftware/go-kit/errors"
)

type ctxKey struct{}
type rawTokenKey struct{}

// WithClaims stores verified claims on the context.
func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// ClaimsFromCtx returns the verified claims, if the request was authenticated.
func ClaimsFromCtx(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(Claims)
	return c, ok
}

// WithRawToken stores the raw bearer token, so a service can forward it on
// service-to-service calls (token passthrough).
func WithRawToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, rawTokenKey{}, token)
}

// RawTokenFromCtx returns the raw bearer token of the authenticated request.
func RawTokenFromCtx(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(rawTokenKey{}).(string)
	return t, ok
}

// OwnerFromCtx is the opaque owner key petstore scopes by and passes to Gleipnir:
// the org when the token carries one, else the subject.
func OwnerFromCtx(ctx context.Context) string {
	c, _ := ClaimsFromCtx(ctx)
	if c.OrgID != "" {
		return c.OrgID
	}
	return c.Subject
}

// Authenticator adapts a Verifier to kitrest's HTTPAuthenticator: it extracts
// the bearer token, verifies it, and injects the claims into the context.
type Authenticator struct {
	verifier *Verifier
}

func NewAuthenticator(v *Verifier) *Authenticator { return &Authenticator{verifier: v} }

func (a *Authenticator) Authenticate(r *http.Request) error {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return apierrors.Unauthorized("missing bearer token")
	}
	raw := strings.TrimPrefix(h, "Bearer ")
	claims, err := a.verifier.Verify(r.Context(), raw)
	if err != nil {
		return apierrors.Unauthorized("invalid token")
	}
	ctx := WithClaims(r.Context(), claims)
	ctx = WithRawToken(ctx, raw)
	*r = *r.WithContext(ctx)
	return nil
}
