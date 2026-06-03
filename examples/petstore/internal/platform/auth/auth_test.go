package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

const (
	testIssuer = "https://aegis.example/realms/petstore"
	testKID    = "test-key-1"
)

var fixedNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// jwksServer returns an httptest server serving a JWKS for key, plus a signer.
func jwksServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: &key.PublicKey, KeyID: testKID, Algorithm: "RS256", Use: "sig"},
	}}
	body, err := json.Marshal(set)
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, key
}

func mint(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(key)
	require.NoError(t, err)
	return signed
}

func newVerifier(srv *httptest.Server) *auth.Verifier {
	return auth.NewVerifier(testIssuer,
		auth.WithJWKSURL(srv.URL),
		auth.WithClock(func() time.Time { return fixedNow }))
}

func TestVerify_Valid(t *testing.T) {
	srv, key := jwksServer(t)
	v := newVerifier(srv)
	token := mint(t, key, jwt.MapClaims{
		"iss":    testIssuer,
		"sub":    "user-123",
		"org_id": "org-abc",
		"exp":    fixedNow.Add(time.Hour).Unix(),
	})

	claims, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "org-abc", claims.OrgID)
}

func TestVerify_WrongIssuer(t *testing.T) {
	srv, key := jwksServer(t)
	v := newVerifier(srv)
	token := mint(t, key, jwt.MapClaims{
		"iss": "https://evil.example/realms/x",
		"sub": "user-123",
		"exp": fixedNow.Add(time.Hour).Unix(),
	})
	_, err := v.Verify(context.Background(), token)
	assert.Error(t, err)
}

func TestVerify_Expired(t *testing.T) {
	srv, key := jwksServer(t)
	v := newVerifier(srv)
	token := mint(t, key, jwt.MapClaims{
		"iss": testIssuer,
		"sub": "user-123",
		"exp": fixedNow.Add(-time.Hour).Unix(),
	})
	_, err := v.Verify(context.Background(), token)
	assert.Error(t, err)
}

func TestVerify_UnknownKID(t *testing.T) {
	srv, key := jwksServer(t)
	v := newVerifier(srv)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": testIssuer, "sub": "u", "exp": fixedNow.Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = "other-key"
	signed, err := tok.SignedString(key)
	require.NoError(t, err)
	_, err = v.Verify(context.Background(), signed)
	assert.Error(t, err)
}

func TestAuthenticator_InjectsClaims(t *testing.T) {
	srv, key := jwksServer(t)
	a := auth.NewAuthenticator(newVerifier(srv))
	token := mint(t, key, jwt.MapClaims{
		"iss": testIssuer, "sub": "user-9", "org_id": "org-9",
		"exp": fixedNow.Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/pets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	require.NoError(t, a.Authenticate(req))

	claims, ok := auth.ClaimsFromCtx(req.Context())
	require.True(t, ok)
	assert.Equal(t, "user-9", claims.Subject)
	assert.Equal(t, "org-9", auth.OwnerFromCtx(req.Context()))
}

func TestAuthenticator_RejectsMissingAndBadTokens(t *testing.T) {
	srv, _ := jwksServer(t)
	a := auth.NewAuthenticator(newVerifier(srv))

	noHeader := httptest.NewRequest(http.MethodGet, "/v1/pets", nil)
	assert.Error(t, a.Authenticate(noHeader))

	bad := httptest.NewRequest(http.MethodGet, "/v1/pets", nil)
	bad.Header.Set("Authorization", "Bearer not-a-jwt")
	assert.Error(t, a.Authenticate(bad))
}
