package client

import (
	"bufio"
	"context"
	"fmt"
	"gobook/pb"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LaptopClient struct {
	service pb.LaptopServiceClient
}

func NewLaptopClient(cc *grpc.ClientConn) *LaptopClient {
	service := pb.NewLaptopServiceClient(cc)
	return &LaptopClient{
		service: service,
	}
}

func (client *LaptopClient) CreateLaptop(laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	res, err := client.service.CreateLaptopService(context.Background(), req)

	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Print("laptop already exist")
		} else {
			log.Print("cannot create laptop")
		}
		return
	}

	log.Printf("create laptop with id: %s", res.Id)
}

func (client *LaptopClient) SearchLaptop(filter *pb.Filter) {
	log.Print("search filter", filter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{Filter: filter}

	stream, err := client.service.SearchLaptopService(ctx, req)
	if err != nil {
		log.Fatal("cannot search laptop", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("cannot receive response: ", err)
		}

		laptop := res.GetLaptop()
		log.Print("- found: ", laptop.GetId())
		log.Print("  + brand: ", laptop.GetBrand())
		log.Print("  + name: ", laptop.GetName())
		log.Print("  + cpu cores: ", laptop.GetCpu().GetNumberCores())
		log.Print("  + cpu min ghz: ", laptop.GetCpu().GetMinGhz())
		log.Print("  + ram: ", laptop.GetRam())
		log.Print("  + price: ", laptop.GetPriceUsd())
	}
}

func (client *LaptopClient) UploadImage(laptopID string, imagePath string) {
	file, err := os.Open(imagePath)

	if err != nil {
		log.Fatal("cannot open file: ", err)
	}

	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.service.UploadImageService(ctx)

	if err != nil {
		log.Fatal("cannot upload image")
	}

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		},
	}

	err = stream.Send(req)

	if err != nil {
		log.Fatal("cannot send image info: ", err)
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("cannot read data to buffer", err)
			break
		}

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		if err != nil {
			log.Fatal("cannot transfer data ", err, stream.RecvMsg(nil))
			break
		}
	}

	res, err := stream.CloseAndRecv()

	if err != nil {
		log.Fatal("cannot receive response ", err)
	}

	log.Printf("image upload with id: %s and size: %d ", res.GetId(), res.GetSize())
}

func (client *LaptopClient) RateLaptop(laptopIDs []string, score []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.service.RateLaptopService(ctx)

	if err != nil {
		return fmt.Errorf("cannot rate laptop: %v", err)
	}

	response := make(chan error)
	go func() {
		res, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			response <- nil
			return
		}
		if err != nil {
			response <- fmt.Errorf("cannot receive stream response: %v", err)
			return
		}

		log.Print("received response:", res)
	}()

	for i, laptopID := range laptopIDs {
		req := &pb.RateLaptopRequest{
			LaptopId: laptopID,
			Score:    score[i],
		}

		err := stream.Send(req)

		if err != nil {
			return fmt.Errorf("cannot send stream request: %v - %v", err, stream.RecvMsg(nil))
		}

		log.Print("send req: ", req)
	}

	err = stream.CloseSend()
	if err != nil {
		return fmt.Errorf("cannot close send %v", err)
	}

	err = <-response
	return err
}
