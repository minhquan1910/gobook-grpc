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
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUploadImageClient(t *testing.T) {
	path := "../tmp"

	laptop := sample.NewLaptop()
	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore(path)

	serverAddress := startTestLaptopServer(t, laptopStore, imageStore, nil)
	time.Sleep(8 * time.Second)
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
}

func TestRateLaptop(t *testing.T) {
	laptopStore := service.NewInMemoryLaptopStore()
	ratingStore := service.NewInMemoryRatingStore()

	serverAddress := startTestLaptopServer(t, laptopStore, nil, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	stream, err := laptopClient.RateLaptopService(context.Background())
	require.NoError(t, err)

	laptop := sample.NewLaptop()
	err = laptopStore.Save(laptop)
	require.NoError(t, err)

	scores := []float64{8, 7.5, 10}
	averages := []float64{8, 7.75, 8.5}

	for i := range scores {
		req := &pb.RateLaptopRequest{
			LaptopId: laptop.Id,
			Score:    scores[i],
		}

		err := stream.Send(req)
		require.NoError(t, err)
	}

	err = stream.CloseSend()
	require.NoError(t, err)
	n := len(averages)
	for idx := range averages {
		res, err := stream.Recv()
		if err == io.EOF {
			require.Equal(t, idx, n)
			return
		}

		require.NoError(t, err)
		require.Equal(t, res.GetLaptopId(), laptop.Id)
		require.Equal(t, averages[idx], res.GetAverageScore())
		require.Equal(t, uint32(idx+1), res.GetRatedCount())
	}
}

func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imageStore service.ImageStore, ratingStore service.RatingStore) string {
	server := service.NewLaptopServer(laptopStore, imageStore, ratingStore)
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
