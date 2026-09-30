package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-32-bytes-long!!!!!!!"

func TestAuthAcceptsValidHS256(t *testing.T) {
	token, err := GenerateAccessToken("u1", "a@x.io", testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	 }
	w, c := probeAuth(t, token)
	if w.Code != http.StatusOK {
		t.Fatalf("valid token rejected: %d", w.Code)
	}
	if c.GetString("user_id") != "u1" {
		t.Fatalf("user_id = %q", c.GetString("user_id"))
	}
	_ = c
}

func TestAuthRejectsNoneAlgorithm(t *testing.T) {
	// `alg: none` must never authenticate, even though no signature check
	// can fail for it — the parser allowlists HS256 only.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{UserID: "u1"}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := probeAuth(t, unsigned)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("none-alg token accepted: %d", w.Code)
	}
}

func TestAuthRejectsWrongSecret(t *testing.T) {
	token, err := GenerateAccessToken("u1", "a@x.io", "wrong-secret-32-bytes-long!!!!!", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := probeAuth(t, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-secret token accepted: %d", w.Code)
	}
}

func TestAuthRejectsExpired(t *testing.T) {
	token, err := GenerateAccessToken("u1", "a@x.io", testSecret, -time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := probeAuth(t, token)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired token accepted: %d", w.Code)
	}
}

func TestParseRefreshTokenRejectsNoneAlgorithm(t *testing.T) {
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone,
		jwt.RegisteredClaims{Subject: "u1"}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRefreshToken(unsigned, testSecret); err == nil {
		t.Fatal("none-alg refresh token accepted")
	}
}

func probeAuth(t *testing.T, token string) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	Auth(testSecret)(c)
	return w, c
}
