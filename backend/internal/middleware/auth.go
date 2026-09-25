package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"pki-certificate-rollover-impact/backend/internal/config"
	"pki-certificate-rollover-impact/backend/internal/repository"
	"pki-certificate-rollover-impact/backend/internal/util"
)

type Claims struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Team        string `json:"team"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}

type Authenticator struct {
	users repository.UserRepository
	cfg   config.Config
}

func NewAuthenticator(users repository.UserRepository, cfg config.Config) *Authenticator {
	return &Authenticator{users: users, cfg: cfg}
}

func (a *Authenticator) Login(c *gin.Context) {
	var request struct {
		Username string `json:"username" binding:"required,min=2,max=80"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.NewError(http.StatusBadRequest, util.CodeValidation, "username and password are required"))
		return
	}
	user, err := a.users.FindByUsername(c.Request.Context(), request.Username)
	if err != nil || !user.Active || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
		util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "invalid username or password"))
		return
	}
	now := time.Now().UTC()
	expires := now.Add(a.cfg.JWTExpiry)
	claims := Claims{UserID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Team: user.Team, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Issuer: a.cfg.JWTIssuer, Subject: user.Username, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.cfg.JWTSecret))
	if err != nil {
		util.Fail(c, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to issue access token", err))
		return
	}
	util.Success(c, http.StatusOK, gin.H{"access_token": token, "expires_at": expires, "user": util.Actor{UserID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Team: user.Team, Role: user.Role}})
}

func (a *Authenticator) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "Bearer token is required"))
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(a.cfg.JWTSecret), nil
		}, jwt.WithIssuer(a.cfg.JWTIssuer), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			util.Fail(c, util.NewError(http.StatusUnauthorized, util.CodeUnauthorized, "access token is invalid or expired"))
			return
		}
		actor := util.Actor{UserID: claims.UserID, Username: claims.Username, DisplayName: claims.DisplayName, Team: claims.Team, Role: claims.Role}
		c.Set("actor", actor)
		c.Next()
	}
}

func Actor(c *gin.Context) (util.Actor, bool) {
	value, exists := c.Get("actor")
	if !exists {
		return util.Actor{}, false
	}
	actor, ok := value.(util.Actor)
	return actor, ok
}

type rateBucket struct {
	Start time.Time
	Count int
}
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	buckets map[string]rateBucket
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{limit: limit, buckets: map[string]rateBucket{}}
}
func (l *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if actor, ok := Actor(c); ok {
			key = scope + ":" + actor.Username
		}
		now := time.Now()
		l.mu.Lock()
		bucket := l.buckets[key]
		if bucket.Start.IsZero() || now.Sub(bucket.Start) >= time.Minute {
			bucket = rateBucket{Start: now}
		}
		bucket.Count++
		l.buckets[key] = bucket
		exceeded := bucket.Count > l.limit
		l.mu.Unlock()
		if exceeded {
			util.Fail(c, util.NewError(http.StatusTooManyRequests, "RATE_LIMITED", "request rate limit exceeded"))
			return
		}
		c.Next()
	}
}
