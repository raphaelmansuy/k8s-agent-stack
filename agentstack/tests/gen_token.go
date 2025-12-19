package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := "change-me-in-production"
	claims := jwt.MapClaims{
		"team_id":    "team_dev",
		"project_id": "prj_dev",
		"scopes":     []string{"agent:*", "project:*", "deployment:*", "api_key:*", "chat:*"},
		"exp":        time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Println("Error signing token:", err)
		return
	}

	fmt.Println(tokenString)
}
