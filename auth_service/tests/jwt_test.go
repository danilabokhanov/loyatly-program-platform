package tests

import (
	userkeys "authservice/auth_storage/user_keys"
	"crypto/md5"
	"encoding/hex"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
)

var setupOnce sync.Once

func setup() {
	setupOnce.Do(func() {
		privateKey := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDWGUzxQT2N0e2P
gbGHE7z6jQHbFwtHzKKPYxjltyTZ2lc5lotl6puNfWOk+tR6JP+saxA9XApIHm60
W2SHgJx57k7v+KV9TrDKInZ2ZJnu6KtXbl6BjhLLWkLxgi2s8WdI3APEqj8OyKzH
TfXCLxvozzWnlRi1mqxBcAOtCUy7fO1bxmhUaYutm6qjzYpxU/BiMIzaGiT/bjvs
URoY6lvs/2/kLBjSmvpv6Ij6IYqWIiYP7PmfijrlxIxaAe/PH7Bq3kNcb/NsDBYz
h+qBwPCq2foqNGKNCdsSu4MdajDkrM+iquj66JRj+PVhO3kvOy6F4rWb6Gi6BsxS
er8idtmdAgMBAAECggEAB3LRh+zABl3UXpWmMc12PlWwLHKXeGmJ00kFkEJfEhZ/
brgBRJmwg4JUAZvCz0kzKu2QMS5MvkXuxR3GJfjU4fM5aOTweB4IDZjO13qvA+CM
3cUUU/xTICLwu1rhPkgXCY2ORQFps/1cXuaR+iTSpEj06ymHSQHd5Bg1A2Omwl0p
PXwU1Ekv7rPcnmFZbQWAKI+eYfm9jP2m0IsEcsd1FABvrPySycKeTaeInMK1l0TY
9HGvLW/29Nvbo/FlYxqNJfFg84ge1sZ6tPFlWyTRh763aEpst/r9/iEA6MilKQhO
XLPZ4P5uoLENGWunXQw1JPh416G1PV1gMisjQkw4yQKBgQDwoT/07vsiNSnER7yj
+Wz+sdipfoR9OL0w1s1epP838RUfbstjVSqPefaEjiWzQNBlMFzxV6W5VwTnhJtq
vry1l+OhxOQSZVv2SBk6Ddj1LAiVAURySW7TKKtG6a/z2lP9+7YO3KTWESTyUPGF
eu11AYkp1HyPm5fo6zt+//uvxQKBgQDjxjfeTf/OBEFb+QP5T0T4M74i7duyzTeV
VW2dP6GU8PZkdfmws2M746RlATsCvlnnk6iRU+LwJevD/UJousnzjgI3yYKvyJDl
qN0N0PY57m+E2IhFnWCDNghW2iKDAqbO+waCwjBkzaQpFtpTrBNA/2Py5UMEX8tr
XDCy0JiH+QKBgBP6PzYVlTH82e/ayNWQQrVOjJ9dyqAe0s44NyqxZiL91/QZHbes
fXEV/hp5NrYQHn0YK885qJ+fkt+pycFt/nrRFmv6zbidQ6pJyBZiye1o73l3dnhK
knHjgXzMr/f921VNzYqkVOcU201m3PZpA0fgjcO0SXcewtjqlrDvjbTFAoGBANJd
j9vRTiCH7ZV0Nyda/uf9Ye4AoJhS0LMrY0GIM0PMCMRf8WwxQcVeSca/jDDMfVxU
E6ulPkNtwoIQtfTkDwDSd1nu0rRnGOwDOaY5CDAY9wZKthEVeL22eZ09egJlwIoJ
bcn2b5uqEaOhZ6M/mci+FyGOfIbdspJFYvTDkxyBAoGAa75AIUVLUiTRroX8iwHm
lQl1jaylVUpbaxVZdXbKrqduz2CTl/mkFs25P6XfG8buK8k7LAQK3CuBIOd+lU71
L1n7E6PPa3Hqj6gl5uzz63zrIuqpLBr1VWgVRQEUBFpKFeUDnoKDMU3gN83gSdLa
ouUsALfTQ3kRTpS6/VxhLwE=
-----END PRIVATE KEY-----`
		publicKey := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1hlM8UE9jdHtj4GxhxO8
+o0B2xcLR8yij2MY5bck2dpXOZaLZeqbjX1jpPrUeiT/rGsQPVwKSB5utFtkh4Cc
ee5O7/ilfU6wyiJ2dmSZ7uirV25egY4Sy1pC8YItrPFnSNwDxKo/Dsisx031wi8b
6M81p5UYtZqsQXADrQlMu3ztW8ZoVGmLrZuqo82KcVPwYjCM2hok/2477FEaGOpb
7P9v5CwY0pr6b+iI+iGKliImD+z5n4o65cSMWgHvzx+wat5DXG/zbAwWM4fqgcDw
qtn6KjRijQnbEruDHWow5KzPoqro+uiUY/j1YTt5LzsuheK1m+hougbMUnq/InbZ
nQIDAQAB
-----END PUBLIC KEY-----`

		if err := os.MkdirAll("credentials", 0755); err != nil {
			panic("Failed to create credentials directory: " + err.Error())
		}

		if err := os.WriteFile(userkeys.JwtPrivateFile, []byte(privateKey), 0644); err != nil {
			panic("Failed to create private key file: " + err.Error())
		}
		if err := os.WriteFile(userkeys.JwtPublicFile, []byte(publicKey), 0644); err != nil {
			panic("Failed to create public key file: " + err.Error())
		}
	})
}

func teardown() {
	_ = os.Remove(userkeys.JwtPrivateFile)
	_ = os.Remove(userkeys.JwtPublicFile)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestGetUserIdByJWT(t *testing.T) {
	userId := uuid.New()
	token := userkeys.GenJWT(userId)
	parsedUserId, valid := userkeys.GetUserIdByJWT(token)

	if !valid || parsedUserId != userId {
		t.Errorf("GetUserIdByJWT failed: got %v, expected %v", parsedUserId, userId)
	}
}

func TestGenJWT(t *testing.T) {
	userId := uuid.New()
	token := userkeys.GenJWT(userId)
	if token == "" {
		t.Errorf("GenJWT returned an empty token")
	}
}

func TestGetPasswordHash(t *testing.T) {
	login := "testUser"
	password := "securePassword"

	expectedHash := md5.Sum([]byte(password + login))
	computedHash := userkeys.GetPasswordHash(login, password)

	if hex.EncodeToString(computedHash[:]) != hex.EncodeToString(expectedHash[:]) {
		t.Errorf("GetPasswordHash mismatch: got %x, expected %x", computedHash, expectedHash)
	}
}
