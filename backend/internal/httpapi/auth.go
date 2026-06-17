package httpapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"campus-market/backend/internal/db"
)

type authClaims struct {
	UserID int64 `json:"uid"`
	Exp    int64 `json:"exp"`
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := passwordDigest([]byte(password), salt)
	return base64.RawURLEncoding.EncodeToString(salt) + "." + base64.RawURLEncoding.EncodeToString(sum), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	actual := passwordDigest([]byte(password), salt)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func passwordDigest(password, salt []byte) []byte {
	digest := append([]byte{}, salt...)
	for i := 0; i < 120000; i++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(digest)
		digest = mac.Sum(nil)
	}
	return digest
}

func (s *Server) issueToken(userID int64) (string, error) {
	claims := authClaims{UserID: userID, Exp: time.Now().Add(7 * 24 * time.Hour).Unix()}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	sig := s.sign(payload)
	return payload + "." + sig, nil
}

func (s *Server) parseToken(token string) (authClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return authClaims{}, errors.New("invalid token")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0]))) {
		return authClaims{}, errors.New("invalid signature")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return authClaims{}, err
	}
	var claims authClaims
	if err := json.Unmarshal(body, &claims); err != nil {
		return authClaims{}, err
	}
	if time.Now().Unix() > claims.Exp {
		return authClaims{}, errors.New("expired token")
	}
	return claims, nil
}

func (s *Server) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func userIDFromPath(path, prefix string) (int64, error) {
	raw := strings.TrimPrefix(path, prefix)
	raw = strings.Trim(raw, "/")
	if strings.Contains(raw, "/") {
		raw = strings.Split(raw, "/")[0]
	}
	return strconv.ParseInt(raw, 10, 64)
}

func randomRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", binary.BigEndian.Uint64(b[:]))
}

func (s *Server) currentUser(r *http.Request) (db.User, bool) {
	value := r.Context().Value(contextUserKey{})
	user, ok := value.(db.User)
	return user, ok
}
