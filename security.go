package main

import (
	"encoding/json"
	"net/http"
	"time"

	"gopkg.in/dgrijalva/jwt-go.v2"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *HttpServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// parse credential
	decoder := json.NewDecoder(r.Body)
	var credentials Credentials
	err := decoder.Decode(&credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// check credentials
	if account, ok := s.Accounts[credentials.Username]; ok {
		if account.Password != credentials.Password {
			http.Error(w, "wrong username/password", http.StatusUnauthorized)
			return
		}
	} else {
		http.Error(w, "wrong username/password", http.StatusUnauthorized)
		return
	}
	// Create the token
	token, err := GenerateToken(&credentials, time.Hour*3, s.HS256key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// copy response to original request
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]string)
	response["token"] = token
	json.NewEncoder(w).Encode(response)
}

func GenerateToken(credentials *Credentials, expiration time.Duration, key []byte) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	// Set some claims
	token.Claims["sub"] = credentials.Username
	token.Claims["it"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(expiration).Unix()
	// Sign and get the complete encoded token as a string
	return token.SignedString(key)
}
