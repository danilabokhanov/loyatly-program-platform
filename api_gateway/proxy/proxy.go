package proxy

import (
	kafka "apigateway/kafka_producer"
	protoauth "apigateway/proto/auth"
	protopromo "apigateway/proto/promo"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const targetBase = "http://nginx:8081"

const promoServiceAddress = "loyalty-service:8083"
const authServiceAddress = "auth-service:8080"

func grpcErrorToHTTP(err error) int {
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}

	var httpStatus int

	switch st.Code() {
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
	case codes.NotFound:
		httpStatus = http.StatusNotFound
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
	case codes.Internal:
		httpStatus = http.StatusInternalServerError
	default:
		httpStatus = http.StatusInternalServerError
	}

	return httpStatus
}

type GrpcClients struct {
	authClient  protoauth.AuthServiceClient
	promoClient protopromo.PromoServiceClient
}

func NewGrpcClients() (*GrpcClients, error) {
	connAuth, err := grpc.Dial(authServiceAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %v", err)
	}
	connPromo, err := grpc.Dial(promoServiceAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to promo service: %v", err)
	}
	return &GrpcClients{authClient: protoauth.NewAuthServiceClient(connAuth), promoClient: protopromo.NewPromoServiceClient(connPromo)}, nil
}

func (g *GrpcClients) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var userCreds protoauth.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&userCreds); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	user, err := g.authClient.Register(ctx, &userCreds)
	if err != nil {
		http.Error(w, err.Error(), grpcErrorToHTTP(err))
		return
	}

	kafka.SendStat("user_registered", user.Id, user.Login)

	w.WriteHeader(http.StatusCreated)
}

func (g *GrpcClients) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var userCreds protoauth.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&userCreds); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	loginResponse, err := g.authClient.Login(ctx, &userCreds)
	if err != nil {
		http.Error(w, err.Error(), grpcErrorToHTTP(err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    loginResponse.Jwt,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		Expires:  time.Now().Add(72 * time.Hour),
	})
	w.WriteHeader(http.StatusOK)
}

func (g *GrpcClients) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("Authorization")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	user, err := g.authClient.GetProfile(ctx, &protoauth.AuthRequest{Jwt: jwt.Value})
	if err != nil {
		http.Error(w, err.Error(), grpcErrorToHTTP(err))
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (g *GrpcClients) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("Authorization")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}
	updateProfileRequest := protoauth.UpdateProfileRequest{NewInfo: &protoauth.User{}}
	if err := json.NewDecoder(r.Body).Decode(updateProfileRequest.NewInfo); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	updateProfileRequest.Jwt = jwt.Value

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	user, err := g.authClient.UpdateProfile(ctx, &updateProfileRequest)
	if err != nil {
		http.Error(w, err.Error(), grpcErrorToHTTP(err))
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (g *GrpcClients) getUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	userIdRaw := chi.URLParam(r, "id")
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	user, err := g.authClient.GetUserById(ctx, &protoauth.UserIdRequest{Id: userId.String()})
	if err != nil {
		http.Error(w, err.Error(), grpcErrorToHTTP(err))
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
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

	var req protopromo.CreatePromoRequest
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
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.GetPromo(ctx, &protopromo.GetPromoRequest{Id: id})
	if err != nil {
		http.Error(w, fmt.Sprintf("Promo not found: %v", err), http.StatusNotFound)
		return
	}

	kafka.SendStat("promo_viewed", userID, id)

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) updatePromoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	var req protopromo.UpdatePromoRequest
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

	_, err = g.promoClient.DeletePromo(ctx, &protopromo.DeletePromoRequest{Id: id, AuthorId: userID})
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

	resp, err := g.promoClient.ListPromos(ctx, &protopromo.ListPromosRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list promos: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) addCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	var req protopromo.AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.AuthorId = userID

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.AddComment(ctx, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add comment: %v", err), grpcErrorToHTTP(err))
		return
	}

	kafka.SendStat("comment_published", userID, resp.Id)

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) getCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.GetComment(ctx, &protopromo.GetCommentRequest{CommentId: id})
	if err != nil {
		http.Error(w, fmt.Sprintf("Comment not found: %v", err), http.StatusNotFound)
		return
	}

	kafka.SendStat("comment_viewed", userID, id)

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) listCommentsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	promoId := chi.URLParam(r, "promo_id")
	page := 1
	pageSize := 10
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := g.promoClient.ListComments(ctx, &protopromo.ListCommentsRequest{
		PromoId:  promoId,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list comments: %v", err), http.StatusInternalServerError)
		return
	}

	for _, comment := range resp.Comments {
		kafka.SendStat("comment_viewed", userID, comment.Id)
	}

	json.NewEncoder(w).Encode(resp)
}

func (g *GrpcClients) promoOnClickHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := g.getUserID(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}

	promoId := chi.URLParam(r, "promo_id")

	kafka.SendStat("promo_click", userID, promoId)

	w.WriteHeader(http.StatusOK)
}

func NewRouter(g *GrpcClients) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/api/v1/register", g.registerUserHandler)
	r.Post("/api/v1/login", g.loginUserHandler)
	r.Get("/api/v1/profile", g.getProfileHandler)
	r.Post("/api/v1/profile", g.updateProfileHandler)
	r.Get("/api/v1/user/{id}", g.getUserInfoHandler)

	r.Post("/api/v1/promos", g.createPromoHandler)
	r.Get("/api/v1/promos/{id}", g.getPromoHandler)
	r.Put("/api/v1/promos/{id}", g.updatePromoHandler)
	r.Delete("/api/v1/promos/{id}", g.deletePromoHandler)
	r.Get("/api/v1/promos", g.listPromosHandler)

	r.Post("/api/v1/comments", g.addCommentHandler)
	r.Get("/api/v1/comments/{id}", g.getCommentHandler)
	r.Get("/api/v1/comments/promo/{promo_id}", g.listCommentsHandler)

	r.Post("/api/v1/on_click/{promo_id}", g.promoOnClickHandler)
	return r
}
