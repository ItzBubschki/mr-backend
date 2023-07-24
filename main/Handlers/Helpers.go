package Handlers

import (
	"context"
	"firebase.google.com/go/v4/auth"
	"log"
	"net/http"
)

func ArrayContains(array []string, value string) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

func AuthorizationWrapper(w http.ResponseWriter, r *http.Request, authHandler *auth.Client) (bool, *auth.Token) {
	if r.Method == http.MethodOptions {
		_, _ = w.Write([]byte("OK"))
		return false, nil
	} else if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false, nil
	}
	idToken := r.Header.Get("Authorization")
	if idToken == "" {
		log.Println("No token found")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false, nil
	}
	token, err := authHandler.VerifyIDToken(context.Background(), idToken)
	if err != nil {
		log.Printf("error verifying ID token: %v\n", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false, nil
	}

	return true, token
}
