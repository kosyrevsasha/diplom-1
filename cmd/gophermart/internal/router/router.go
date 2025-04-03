package router

import (
	"context"
	"diplom-1/cmd/gophermart/internal/accrual"
	"diplom-1/cmd/gophermart/internal/config"
	"diplom-1/cmd/gophermart/internal/handler"
	"diplom-1/cmd/gophermart/internal/repository"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"time"
)

type (
	responseData struct {
		status int
		size   int
	}

	Claims struct {
		jwt.RegisteredClaims
		UserID int
	}
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			handler.MakeErrResponse(&w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		token := strings.Replace(auth, "Bearer ", "", 1)
		if token == "" {
			handler.MakeErrResponse(&w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		claims := &Claims{}
		jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(config.Secretkey), nil
		})
		fmt.Println(claims.ExpiresAt)
		if claims.ExpiresAt.Time.Before(time.Now()) {
			handler.MakeErrResponse(&w, http.StatusUnauthorized, "Token expired")
			return
		}

		ctx := context.WithValue(r.Context(), config.ProcessConfig.TokenKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func BuildRouter(db repository.Repository, worker *accrual.Worker) chi.Router {
	r := chi.NewRouter()

	// public routes
	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) { handler.Register(db, w, r) })
		r.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) { handler.Login(db, w, r) })
	})

	// auth requires routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Route("/api", func(r chi.Router) {
			r.Route("/user", func(r chi.Router) {
				r.Post("/orders", func(w http.ResponseWriter, r *http.Request) { handler.ProcessOrder(db, w, r, worker) })
				r.Get("/orders", func(w http.ResponseWriter, r *http.Request) { handler.UserOrders(db, w, r) })
				r.Get("/balance", func(w http.ResponseWriter, r *http.Request) { handler.UserBalance(db, w, r) })
				r.Post("/balance/withdraw", func(w http.ResponseWriter, r *http.Request) { handler.Withdraw(db, w, r) })
				r.Get("/withdrawals", func(w http.ResponseWriter, r *http.Request) { handler.Withdraws(db, w, r) })
			})
		})
	})
	return r
}
