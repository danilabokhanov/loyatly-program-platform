package authhandlers

import (
	usermodel "authservice/auth_storage/user_model"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

var storageManager usermodel.StorageManager

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var userCreds usermodel.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&userCreds); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	user, err := storageManager.CreateUser(userCreds.Login, userCreds.Password, userCreds.Email, userCreds.IsCompany)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create user: %v", err), http.StatusInternalServerError)
		return
	}
	if user.Login == "" {
		http.Error(w, "Invalid credentials format or user has already exists", http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var userCreds usermodel.UserCreds
	if err := json.NewDecoder(r.Body).Decode(&userCreds); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	if userCreds.Login == "" || userCreds.Password == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	jwt, err := storageManager.GetJWTByCredentials(userCreds.Login, userCreds.Password)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}
	if jwt == "" {
		http.Error(w, "Invalid user credentials", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    jwt,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		Expires:  time.Now().Add(72 * time.Hour),
	})
	w.WriteHeader(http.StatusOK)
}

func getProfileHandler(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("Authorization")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}
	user, err := storageManager.GetUserByJWT(jwt.Value)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user.Login == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	jwt, err := r.Cookie("Authorization")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
		return
	}
	var user usermodel.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, fmt.Sprintf("Invalid format: %v", err), http.StatusBadRequest)
		return
	}
	user, err = storageManager.UpdateUserByJWT(jwt.Value, user)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user.Login == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	userIdRaw := chi.URLParam(r, "id")
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusInternalServerError)
		return
	}
	user, err := storageManager.GetUserById(userId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}
	if user.Login == "" {
		http.Error(w, "User not Found", http.StatusNotFound)
		return
	}
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func NewRouter(sm usermodel.StorageManager) *chi.Mux {
	storageManager = sm
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/api/v1/register", registerUserHandler)
	r.Post("/api/v1/login", loginUserHandler)
	r.Get("/api/v1/profile", getProfileHandler)
	r.Post("/api/v1/profile", updateProfileHandler)
	r.Get("/api/v1/user/{id}", getUserInfoHandler)
	return r
}
