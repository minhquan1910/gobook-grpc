package service

import (
	"context"
	"gobook/pb"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestAuthLogin(t *testing.T) {
	username := "quan"
	password := "secret"
	role := "employer"
	user, err := NewUser(username, password, role)
	require.NoError(t, err)
	require.NotNil(t, user)

	userStore := NewInMemoryUserStore()
	err = userStore.Save(user)
	require.NoError(t, err)

	key := "e8c17fd65e37a83147f021726921fe75"
	jwtManager := NewJWTToken(key)
	serverAddress := startTestAuthServer(t, userStore, jwtManager)
	client := newTestAuthClient(t, serverAddress)
	req := &pb.LoginRequest{
		Username: username,
		Password: password,
	}
	res, err := client.Login(context.Background(), req)
	require.NoError(t, err)

	accessToken := res.AccessToken
	payload, err := jwtManager.VerifyToken(accessToken)
	require.NoError(t, err)
	require.Equal(t, payload.Username, username)
}

func startTestAuthServer(t *testing.T, userStore UserStore, jwtManager JWTManager) string {
	authServer := NewAuthServer(userStore, jwtManager)

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener)

	return listener.Addr().String()
}

func newTestAuthClient(t *testing.T, address string) pb.AuthServiceClient {

	cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return pb.NewAuthServiceClient(cc)
}
