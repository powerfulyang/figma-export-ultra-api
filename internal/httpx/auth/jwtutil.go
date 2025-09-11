package auth

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"fiber-ent-apollo-pg/internal/config"
)

// Claims represents JWT claims used by this service.
type Claims struct {
	Kind     string   `json:"kind"`
	Roles    []string `json:"roles,omitempty"`
	DeviceID string   `json:"device_id,omitempty"`
	jwt.RegisteredClaims
}

type tokenKeys struct {
	method    jwt.SigningMethod
	signKey   any
	verifyKey any
}

func loadKeys(cfg *config.Config) (*tokenKeys, error) {
	algo := cfg.JWT.Algo
	switch algo {
	case "RS256":
		privPem := cfg.JWT.RSPrivateKey
		pubPem := cfg.JWT.RSPublicKey
		if privPem == "" || pubPem == "" {
			return &tokenKeys{method: jwt.SigningMethodHS256, signKey: []byte(cfg.JWT.HSSecret), verifyKey: []byte(cfg.JWT.HSSecret)}, nil
		}
		priv, err := parseRSAPrivateKeyFromPEM([]byte(privPem))
		if err != nil {
			return nil, err
		}
		pub, err := parseRSAPublicKeyFromPEM([]byte(pubPem))
		if err != nil {
			return nil, err
		}
		return &tokenKeys{method: jwt.SigningMethodRS256, signKey: priv, verifyKey: pub}, nil
	case "HS256":
		return &tokenKeys{method: jwt.SigningMethodHS256, signKey: []byte(cfg.JWT.HSSecret), verifyKey: []byte(cfg.JWT.HSSecret)}, nil
	default:
		return nil, errors.New("unsupported JWT_ALGO")
	}
}

func parseRSAPrivateKeyFromPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid RSA private PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}
	k8, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, err
	}
	switch k := k8.(type) {
	case *rsa.PrivateKey:
		return k, nil
	default:
		return nil, errors.New("unsupported PKCS8 private key type")
	}
}

func parseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid RSA public PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return k, nil
	case *ecdsa.PublicKey:
		return nil, errors.New("ECDSA not supported here")
	default:
		return nil, errors.New("unsupported public key type")
	}
}

// SignAccess issues a short-lived access token.
func SignAccess(cfg *config.Config, sub string, kind string, roles []string, deviceID string) (string, string, error) {
	keys, err := loadKeys(cfg)
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	jti := uuid.NewString()
	claims := &Claims{
		Kind:     kind,
		Roles:    roles,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.JWT.Issuer,
			Audience:  jwt.ClaimStrings{cfg.JWT.Audience},
			Subject:   sub,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWT.AccessMin) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(keys.method, claims)
	s, err := token.SignedString(keys.signKey)
	return s, jti, err
}

// SignRefresh issues a long-lived refresh token.
func SignRefresh(cfg *config.Config, sub string, kind string, deviceID string) (string, string, error) {
	keys, err := loadKeys(cfg)
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	jti := uuid.NewString()
	claims := &Claims{
		Kind:     kind,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.JWT.Issuer,
			Audience:  jwt.ClaimStrings{cfg.JWT.Audience},
			Subject:   sub,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWT.RefreshDays) * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(keys.method, claims)
	s, err := token.SignedString(keys.signKey)
	return s, jti, err
}

// ParseAndValidate verifies a token string and returns claims.
func ParseAndValidate(cfg *config.Config, tokenStr string) (*Claims, error) {
	keys, err := loadKeys(cfg)
	if err != nil {
		return nil, err
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{keys.method.Alg()}))
	tok, err := parser.ParseWithClaims(tokenStr, &Claims{}, func(_ *jwt.Token) (interface{}, error) { return keys.verifyKey, nil })
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

// SetRefreshCookie sets the refresh token as HttpOnly cookie.
func SetRefreshCookie(c *fiber.Ctx, token string, ttlDays int) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   ttlDays * 24 * 60 * 60,
	})
}

// ClearRefreshCookie clears the refresh cookie.
func ClearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: "refresh_token", Value: "", MaxAge: -1, Path: "/"})
}
