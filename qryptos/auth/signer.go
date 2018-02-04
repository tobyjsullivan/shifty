package main

import (
	"time"
	"github.com/dgrijalva/jwt-go"
	"strconv"
	"os"
)

const (
	path = "/orders"
)

var (
	tokenId = os.Getenv("TOKEN_ID")
	userSecret = os.Getenv("USER_SECRET")
	nonce = time.Now().UnixNano() / 1000000
)

func main() {
	println("Token ID:", tokenId)
	println("Nonce:", nonce)


	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"path":     path,
		"nonce":    strconv.FormatInt(nonce, 10),
		"token_id": tokenId,
	})

	tokenString, err := token.SignedString([]byte(userSecret))
	if err != nil {
		panic(err.Error())
	}

	println()
	println("JWT:", tokenString)
}
