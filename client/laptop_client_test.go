package client

import (
	"bufio"
	"context"
	"fmt"
	"gobook/pb"
	"gobook/sample"
	"gobook/service"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUploadImageClient(t *testing.T) {
	path := "../tmp"

	laptop := sample.NewLaptop()
	laptopStore := service.NewInMemoryLaptopStore()

	serverAddress := startTestLaptopServer(t, laptopStore, nil)
	laptopClient := newTestLaptopClient(t, serverAddress)

	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	stream, err := laptopClient.UploadImageService(context.Background())
	require.NoError(t, err)

	filePath := fmt.Sprintf("%s/laptop.jpg", path)
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptop.Id,
				ImageType: filepath.Ext(filePath),
			},
		},
	}

	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		require.NoError(t, err)

		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		size += n

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.EqualValues(t, res.GetSize(), size)

	saveImagePath := fmt.Sprintf("%s/%s%s", path, res.GetId(), filepath.Ext(filePath))

	require.FileExists(t, saveImagePath)
	require.NoError(t, os.Remove(saveImagePath))
}

func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imageStore service.ImageStore) string {
	server := service.NewLaptopServer(laptopStore, imageStore, nil)
	require.NotNil(t, server)

	grpcServer := grpc.NewServer()
	require.NotNil(t, grpcServer)

	pb.RegisterLaptopServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener)

	return listener.Addr().String()
}

func newTestLaptopClient(t *testing.T, serverAddress string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client := pb.NewLaptopServiceClient(conn)
	return client
}
