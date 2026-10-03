package auth

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const testSecret = "test-secret"

func newTestService(t *testing.T) *AuthService {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewAuthService(&config.Config{JWTSecret: testSecret, PasswordPepper: "pepper"}, db)
}

func signToken(t *testing.T, method jwt.SigningMethod, key interface{}, claims jwt.MapClaims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestNeedsBootstrap(t *testing.T) {
	s := newTestService(t)
	if !s.NeedsBootstrap() {
		t.Fatal("an empty database must need bootstrap")
	}
	if err := s.Register("admin", "hunter2hunter2"); err != nil {
		t.Fatal(err)
	}
	if s.NeedsBootstrap() {
		t.Fatal("bootstrap must be done once a user exists")
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	s := newTestService(t)
	if err := s.Register("admin", "first-password"); err != nil {
		t.Fatal(err)
	}
	if err := s.Register("admin", "second-password"); err == nil {
		t.Fatal("registering the same username twice must fail")
	}
}

func TestLoginIssuesValidToken(t *testing.T) {
	s := newTestService(t)
	s.Register("admin", "correct-password")

	token, err := s.Login("admin", "correct-password")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.ValidateToken(token)
	if err != nil {
		t.Fatalf("freshly issued token rejected: %v", err)
	}
	if claims["username"] != "admin" {
		t.Errorf("username claim = %v, want admin", claims["username"])
	}
	if _, ok := claims["user_id"].(float64); !ok {
		t.Errorf("user_id claim missing or wrong type: %v", claims["user_id"])
	}

	exp, _ := claims["exp"].(float64)
	remaining := time.Until(time.Unix(int64(exp), 0))
	if remaining < 71*time.Hour || remaining > 72*time.Hour+time.Minute {
		t.Errorf("token lifetime = %v, want about 72h", remaining)
	}
}

func TestLoginFailuresShareOneMessage(t *testing.T) {
	s := newTestService(t)
	s.Register("admin", "correct-password")

	_, wrongPassword := s.Login("admin", "wrong-password")
	_, unknownUser := s.Login("nobody", "correct-password")
	if wrongPassword == nil || unknownUser == nil {
		t.Fatal("both attempts must fail")
	}
	if wrongPassword.Error() != unknownUser.Error() {
		t.Errorf("messages differ and leak which usernames exist: %q vs %q", wrongPassword, unknownUser)
	}
}

func TestLoginDependsOnPepper(t *testing.T) {
	s := newTestService(t)
	s.Register("admin", "correct-password")

	s.config.PasswordPepper = "another-pepper"
	if _, err := s.Login("admin", "correct-password"); err == nil {
		t.Fatal("a changed pepper must invalidate stored passwords")
	}
}

func TestValidateTokenRejects(t *testing.T) {
	s := newTestService(t)
	valid := jwt.MapClaims{"user_id": 1, "username": "admin", "exp": time.Now().Add(time.Hour).Unix()}
	expired := jwt.MapClaims{"user_id": 1, "username": "admin", "exp": time.Now().Add(-time.Minute).Unix()}

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not.a.jwt"},
		{"expired", signToken(t, jwt.SigningMethodHS256, []byte(testSecret), expired)},
		{"wrong secret", signToken(t, jwt.SigningMethodHS256, []byte("other-secret"), valid)},
		{"alg none", signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := s.ValidateToken(tt.token); err == nil {
				t.Error("token must be rejected")
			}
		})
	}
}

func TestValidateTokenAcceptsValid(t *testing.T) {
	s := newTestService(t)
	token := signToken(t, jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{
		"user_id": 1, "username": "admin", "exp": time.Now().Add(time.Hour).Unix(),
	})
	if _, err := s.ValidateToken(token); err != nil {
		t.Errorf("valid token rejected: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	s := newTestService(t)
	s.Register("admin", "old-password")

	if err := s.ChangePassword(1, "not-the-password", "new-password"); err == nil {
		t.Fatal("a wrong current password must be refused")
	}
	if _, err := s.Login("admin", "old-password"); err != nil {
		t.Fatalf("failed attempt must not change the password: %v", err)
	}

	if err := s.ChangePassword(1, "old-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login("admin", "old-password"); err == nil {
		t.Error("old password must stop working")
	}
	if _, err := s.Login("admin", "new-password"); err != nil {
		t.Errorf("new password must work: %v", err)
	}

	if err := s.ChangePassword(999, "x", "y"); err == nil {
		t.Error("unknown user must be refused")
	}
}

func TestMiddleware(t *testing.T) {
	s := newTestService(t)
	s.Register("admin", "correct-password")
	token, _ := s.Login("admin", "correct-password")

	e := echo.New()
	handler := s.Middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, c.Get("username").(string))
	})

	call := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler(e.NewContext(req, rec))
		return rec
	}

	if rec := call(nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("no cookie: status = %d, want 401", rec.Code)
	}
	if rec := call(&http.Cookie{Name: "token", Value: "garbage"}); rec.Code != http.StatusUnauthorized {
		t.Errorf("bad cookie: status = %d, want 401", rec.Code)
	}
	rec := call(&http.Cookie{Name: "token", Value: token})
	if rec.Code != http.StatusOK || rec.Body.String() != "admin" {
		t.Errorf("valid cookie: status = %d body = %q, want 200 admin", rec.Code, rec.Body.String())
	}
}
