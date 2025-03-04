package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const targetBase = "http://auth-service:8080"

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(targetBase + r.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("API Gateway failed: %v", err), http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest(r.Method, target.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	fmt.Println("Cookie1", r.Header.Get("Cookie"))
	if cookies := r.Header.Get("Cookie"); cookies != "" {
		req.Header.Set("Cookie", cookies)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error forwarding request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	fmt.Println("Cookie2", resp.Cookies())
	for _, cookie := range resp.Cookies() {
		http.SetCookie(w, cookie)
	}

	io.Copy(w, resp.Body)
}

func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/v1/register", proxyHandler)
	r.Post("/api/v1/login", proxyHandler)
	r.Get("/api/v1/profile", proxyHandler)
	r.Post("/api/v1/profile", proxyHandler)
	r.Get("/api/v1/user/{id}", proxyHandler)
	return r
}
