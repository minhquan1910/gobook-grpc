package service

import (
	"context"
	"net"
	"testing"
	"time"

	"gobook/pb"
	"gobook/sample"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientCreateLaptop(t *testing.T) {
	laptopStore := NewInMemoryLaptopStore()

	address := startTestLaptopServer(t, laptopStore)
	time.Sleep(time.Second * 20)
	client := newTestLaptopClient(t, address)
	laptop := sample.NewLaptop()
	expectId := laptop.Id
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	res, err := client.CreateLaptopService(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, expectId, res.Id)

	savedLaptop := laptopStore.Find(res.Id)

	require.NotNil(t, savedLaptop)
}

func startTestLaptopServer(t *testing.T, laptopStore LaptopStore) string {
	laptopServer := NewLaptopServer(laptopStore, nil, nil)

	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener)

	return listener.Addr().String()
}

func newTestLaptopClient(t *testing.T, address string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn)
}
