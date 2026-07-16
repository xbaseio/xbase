package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xbaseio/xbase/utils/jwt"
)

type response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

var (
	auth    *jwt.JWT
	payload = jwt.Payload{
		"uid":     1,
		"account": "xbase",
	}
)

func init() {
	var err error
	auth, err = jwt.NewJWT(
		jwt.WithIssuer("backend"),
		jwt.WithSignAlgorithm(jwt.HS256),
		jwt.WithSecretKey("replace-with-a-secure-secret"),
		jwt.WithValidDuration(3600),
		jwt.WithLookupLocations("header:Authorization"),
		jwt.WithIdentityKey("uid"),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Status: status, Message: message, Data: data})
}

func authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, err := auth.Http().Middleware(r)
		if err != nil {
			message := "unauthorized"
			switch {
			case jwt.IsInvalidToken(err):
				message = "token is invalid"
			case jwt.IsExpiredToken(err):
				message = "token is expired"
			case jwt.IsMissingToken(err):
				message = "token is missing"
			case jwt.IsAuthElsewhere(err):
				message = "auth elsewhere"
			}
			writeJSON(w, http.StatusUnauthorized, message, nil)
			return
		}

		next.ServeHTTP(w, request)
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, _ *http.Request) {
		token, err := auth.GenerateToken(payload)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, "login success", token)
	})

	mux.HandleFunc("DELETE /logout", func(w http.ResponseWriter, r *http.Request) {
		if err := auth.Http().DestroyToken(r); err != nil {
			writeJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, "logout success", nil)
	})

	mux.HandleFunc("PUT /refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Http().RefreshToken(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, "refresh success", token)
	})

	mux.Handle("GET /profile", authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := auth.Http().ExtractPayload(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, "get profile success", info)
	})))

	log.Println("JWT example listening on http://127.0.0.1:8888")
	log.Fatal(http.ListenAndServe(":8888", mux))
}
