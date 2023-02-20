package service

import (
	"context"
	"gobook/pb"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	userStore  UserStore
	jwtManager JWTManager
}

func NewAuthServer(userStore UserStore, jwtManager JWTManager) *AuthServer {
	return &AuthServer{
		userStore:  userStore,
		jwtManager: jwtManager,
	}
}

func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := server.userStore.Find(req.GetUsername())

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cannot find user %v", err)
	}

	if user.IsPasswordCorrect(req.GetPassword()) {
		return nil, status.Errorf(codes.Internal, "wrong password")
	}

	token, _, err := server.jwtManager.CreateToken(req.GetUsername(), 15*time.Minute)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create token")
	}
	res := &pb.LoginResponse{
		AccessToken: token,
	}

	return res, nil
}
