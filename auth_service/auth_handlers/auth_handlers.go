package authhandlers

import (
	usermodel "authservice/auth_storage/user_model"
	pb "authservice/proto/auth"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// var storageManager usermodel.StorageManager

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	storageManager usermodel.StorageManager
}

func NewAuthServer(storageManager usermodel.StorageManager) *AuthServer {
	return &AuthServer{storageManager: storageManager}
}

func ConvertUserToProto(src usermodel.User) *pb.User {
	dst := &pb.User{}

	if src.ID != uuid.Nil {
		dst.Id = src.ID.String()
	}
	if src.FirstName != "" {
		dst.FirstName = src.FirstName
	}
	if src.SecondName != "" {
		dst.SecondName = src.SecondName
	}
	if !src.BirthDate.IsZero() {
		dst.BirthDate = src.BirthDate.Format(time.RFC3339)
	}
	if src.Email != "" {
		dst.Email = src.Email
	}
	if src.PhoneNumber != "" {
	}
	dst.IsCompany = src.IsCompany
	if !src.CreationDate.IsZero() {
		dst.CreationDate = src.CreationDate.Format(time.RFC3339)
	}
	if !src.UpdateDate.IsZero() {
		dst.UpdateDate = src.UpdateDate.Format(time.RFC3339)
	}
	if src.Login != "" {
		dst.Login = src.Login
	}
	return dst
}

const timeLayout = time.RFC3339

func ConvertProtoToUser(src *pb.User) usermodel.User {
	if src == nil {
		return usermodel.User{}
	}

	var dst usermodel.User

	if src.Id != "" {
		if parsedID, err := uuid.Parse(src.Id); err == nil {
			dst.ID = parsedID
		}
	}

	if src.FirstName != "" {
		dst.FirstName = src.FirstName
	}
	if src.SecondName != "" {
		dst.SecondName = src.SecondName
	}

	if src.BirthDate != "" {
		if t, err := time.Parse(timeLayout, src.BirthDate); err == nil {
			dst.BirthDate = t
		}
	}

	if src.Email != "" {
		dst.Email = src.Email
	}
	if src.PhoneNumber != "" {
		dst.PhoneNumber = src.PhoneNumber
	}

	dst.IsCompany = src.IsCompany

	if src.CreationDate != "" {
		if t, err := time.Parse(timeLayout, src.CreationDate); err == nil {
			dst.CreationDate = t
		}
	}

	if src.UpdateDate != "" {
		if t, err := time.Parse(timeLayout, src.UpdateDate); err == nil {
			dst.UpdateDate = t
		}
	}

	if src.Login != "" {
		dst.Login = src.Login
	}

	return dst
}

func (s *AuthServer) Register(ctx context.Context, creds *pb.UserCreds) (*pb.User, error) {
	user, err := s.storageManager.CreateUser(creds.Login, creds.Password, creds.Email, creds.IsCompany)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create user: %v", err)
	}
	if user.Login == "" {
		return nil, status.Error(codes.AlreadyExists, "user already exists or invalid credentials")
	}
	return ConvertUserToProto(user), nil
}

func (s *AuthServer) Login(ctx context.Context, creds *pb.UserCreds) (*pb.LoginResponse, error) {
	if creds.Login == "" || creds.Password == "" {
		return nil, status.Error(codes.Unauthenticated, "missing login or password")
	}
	jwt, err := s.storageManager.GetJWTByCredentials(creds.Login, creds.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get JWT: %v", err)
	}
	if jwt == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return &pb.LoginResponse{Jwt: jwt}, nil
}

func (s *AuthServer) GetProfile(ctx context.Context, req *pb.AuthRequest) (*pb.User, error) {
	if req.Jwt == "" {
		return nil, status.Error(codes.Unauthenticated, "missing JWT")
	}
	user, err := s.storageManager.GetUserByJWT(req.Jwt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user.Login == "" {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return ConvertUserToProto(user), nil
}

func (s *AuthServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.User, error) {
	if req.Jwt == "" {
		return nil, status.Error(codes.Unauthenticated, "missing JWT")
	}
	user := ConvertProtoToUser(req.NewInfo)
	fmt.Println(user)
	user, err := s.storageManager.UpdateUserByJWT(req.Jwt, user)
	fmt.Println(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}
	if user.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid user data")
	}
	return ConvertUserToProto(user), nil
}

func (s *AuthServer) GetUserById(ctx context.Context, req *pb.UserIdRequest) (*pb.User, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}
	user, err := s.storageManager.GetUserById(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user.Login == "" {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return ConvertUserToProto(user), nil
}
