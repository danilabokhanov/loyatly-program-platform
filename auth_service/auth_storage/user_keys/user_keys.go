package userkeys

import (
	"crypto/md5"
	"crypto/rsa"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

const (
	Md5Len         = 16
	JwtPublicFile  = "credentials/jwt_public_key.txt"
	JwtPrivateFile = "credentials/jwt_private_key.txt"
)

var (
	jwtPublic  *rsa.PublicKey
	jwtPrivate *rsa.PrivateKey
	once       sync.Once
)

func getJwtKeys() (*rsa.PublicKey, *rsa.PrivateKey) {
	once.Do(func() {
		var err error
		private, err := os.ReadFile(JwtPrivateFile)
		if err != nil {
			log.Fatalf("Error reading jwt private key: %v", err)
		}
		public, err := os.ReadFile(JwtPublicFile)
		if err != nil {
			log.Fatalf("Error reading jwt public key: %v", err)
		}
		jwtPrivate, err = jwt.ParseRSAPrivateKeyFromPEM(private)
		if err != nil {
			log.Fatalf("Error parsing jwt private key: %v", err)
		}
		jwtPublic, err = jwt.ParseRSAPublicKeyFromPEM(public)
		if err != nil {
			log.Fatalf("Error parsing jwt public key: %v", err)
		}
	})
	return jwtPublic, jwtPrivate
}

func GetUserIdByJWT(tokenRaw string) (uuid.UUID, bool) {
	token, err := jwt.Parse(tokenRaw, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("Wrong signing method")
		}
		jwtPublic, _ := getJwtKeys()
		return jwtPublic, nil
	})
	if err != nil {
		log.Println("Error fetching jwt token: %v", err)
		return uuid.Nil, false
	}
	if !token.Valid {
		return uuid.Nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, false
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return uuid.Nil, false
	}
	expirationTime := time.Unix(int64(exp), 0)
	if time.Now().After(expirationTime) {
		return uuid.Nil, false
	}
	userIdraw, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, false
	}
	userId, err := uuid.Parse(userIdraw)
	if err != nil {
		return uuid.Nil, false
	}
	return userId, true
}

func GenJWT(userId uuid.UUID) string {
	claims := &jwt.MapClaims{
		"user_id": userId.String(),
		"iss":     "auth-service",
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
		"iat":     time.Now().Unix(),
	}
	_, jwtPrivate := getJwtKeys()
	tokenRaw, err := jwt.NewWithClaims(jwt.SigningMethodRS256,
		claims).SignedString(jwtPrivate)
	if err != nil {
		log.Fatalf("Error generating jwt token: %v", err)
	}
	return tokenRaw
}

func GetPasswordHash(login string, password string) [Md5Len]byte {
	hashGenerator := md5.New()
	_, err := io.WriteString(hashGenerator, password+login)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
	}
	hash := hashGenerator.Sum(nil)
	var buff [Md5Len]byte
	copy(buff[:], hash)
	return buff
}
