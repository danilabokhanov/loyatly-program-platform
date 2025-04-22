package proxy

import (
	proto "apigateway/proto"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"google.golang.org/grpc"
)

const targetBase = "http://auth-service:8080"
const promoServiceAddress = "loyalty-service:8083"

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(targetBase + r.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("API GrpcClients failed: %v", err), http.StatusBadRequest)
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

type GrpcClients struct {
	promoClient proto.PromoServiceClient
}

func NewGrpcClients() (*GrpcClients, error) {
	conn, err := grpc.Dial(promoServiceAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to promo service: %v", err)
	}
	return &GrpcClients{promoClient: proto.NewPromoServiceClient(conn)}, nil
}

func (g *GrpcClients) getUserID(r *http.Request) (string, error) {
	jwtToken := ""
	cookie, err := r.Cookie("Authorization")
	if err == nil {
		jwtToken = cookie.Value
	}

	if jwtToken == "" {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read request body: %v", err)
		}

		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		var loginData struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}

		if err := json.Unmarshal(bodyBytes, &loginData); err != nil {
			return "", fmt.Errorf("failed to decode login data: %v", err)
		}

		loginURL := fmt.Sprintf("%s/api/v1/login", targetBase)
		loginResp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil || loginResp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to authenticate")
		}
		defer loginResp.Body.Close()

		for _, cookie := range loginResp.Cookies() {
			if cookie.Name == "Authorization" {
				jwtToken = cookie.Value
				break
			}
		}

		if jwtToken == "" {
			return "", fmt.Errorf("JWT not found in response")
		}
	}

	profileURL := fmt.Sprintf("%s/api/v1/profile", targetBase)
	req, _ := http.NewRequest("GET", profileURL, nil)
	req.Header.Set("Cookie", "Authorization="+jwtToken)

	profileResp, err := http.DefaultClient.Do(req)
	if err != nil || profileResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user profile")
	}
	defer profileResp.Body.Close()

	var profile struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		return "", fmt.Errorf("failed to decode user profile response")
	}

	return profile.ID, nil
}

func (g *GrpcClients) createPromoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	var req proto.CreatePromoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.AuthorId = userID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.CreatePromo(ctx, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create promo: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) getPromoHandler(w http.ResponseWriter, r *http.Request) {
	_, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.GetPromo(ctx, &proto.GetPromoRequest{Id: id})
	if err != nil {
		http.Error(w, fmt.Sprintf("Promo not found: %v", err), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) updatePromoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	var req proto.UpdatePromoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.AuthorId = userID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.UpdatePromo(ctx, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update promo: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) deletePromoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = g.promoClient.DeletePromo(ctx, &proto.DeletePromoRequest{Id: id, AuthorId: userID})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete promo: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GrpcClients) listPromosHandler(w http.ResponseWriter, r *http.Request) {
	_, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.ListPromos(ctx, &proto.ListPromosRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list promos: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func NewRouter(g *GrpcClients) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/v1/register", proxyHandler)
	r.Post("/api/v1/login", proxyHandler)
	r.Get("/api/v1/profile", proxyHandler)
	r.Post("/api/v1/profile", proxyHandler)
	r.Get("/api/v1/user/{id}", proxyHandler)

	r.Post("/api/v1/promos", g.createPromoHandler)
	r.Get("/api/v1/promos/{id}", g.getPromoHandler)
	r.Put("/api/v1/promos/{id}", g.updatePromoHandler)
	r.Delete("/api/v1/promos/{id}", g.deletePromoHandler)
	r.Get("/api/v1/promos", g.listPromosHandler)
	return r
}
