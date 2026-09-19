package auth

import (
	"fmt"
	"net/http"
	"time"

	"gitpatrol/internal/config"
	"gitpatrol/internal/database"
	"gitpatrol/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	config *config.Config
	db     *database.DB
}

func NewAuthService(cfg *config.Config, db *database.DB) *AuthService {
	return &AuthService{
		config: cfg,
		db:     db,
	}
}

func (s *AuthService) NeedsBootstrap() bool {
        var count int
        err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
        if err != nil {
                return false
        }
        return count == 0
}

func (s *AuthService) Register(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password+s.config.PasswordPepper), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, string(hashedPassword))
	return err
}

func (s *AuthService) Login(username, password string) (string, error) {
	var user models.User
	err := s.db.QueryRow("SELECT id, username, password_hash FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return "", fmt.Errorf("invalid username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password+s.config.PasswordPepper))
	if err != nil {
		return "", fmt.Errorf("invalid username or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *AuthService) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("token")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		}

		claims, err := s.ValidateToken(cookie.Value)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		}

		c.Set("user_id", int(claims["user_id"].(float64)))
		c.Set("username", claims["username"].(string))

		return next(c)
	}
}

func (s *AuthService) ChangePassword(userID int, oldPassword, newPassword string) error {
	var hash string
	if err := s.db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&hash); err != nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword+s.config.PasswordPepper)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword+s.config.PasswordPepper), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), userID)
	return err
}
