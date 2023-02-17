package service

import (
	"bytes"
	"context"
	"errors"
	"gobook/pb"
	"io"
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxImageSize = 1 << 20

type LaptopServer struct {
	pb.UnimplementedLaptopServiceServer
	laptopStore LaptopStore
	imageStore  ImageStore
	ratingStore RatingStore
}

func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, ratingStore RatingStore) pb.LaptopServiceServer {
	return &LaptopServer{
		laptopStore: laptopStore,
		imageStore:  imageStore,
		ratingStore: ratingStore,
	}
}

func (server *LaptopServer) CreateLaptopService(
	ctx context.Context,
	req *pb.CreateLaptopRequest,
) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()

	log.Printf("create a new laptop id: %s", laptop.Id)

	if len(laptop.Id) > 0 {
		//Check if id is valid or not
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "id is invalid %v", err)
		}
	} else {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot random id %v", err)

		}
		laptop.Id = id.String()
	}

	// Save the laptop to in memory store
	err := server.laptopStore.Save(laptop)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}
		return nil, status.Errorf(code, "cannot save laptop to in-memory store %v", err)
	}

	rsp := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}

	return rsp, nil
}

func (server *LaptopServer) SearchLaptopService(
	req *pb.SearchLaptopRequest,
	stream pb.LaptopService_SearchLaptopServiceServer,
) error {
	filter := req.GetFilter()

	log.Printf("receive a search-laptop request with filter: %v", filter)

	err := server.laptopStore.Search(
		stream.Context(),
		filter,
		func(laptop *pb.Laptop) error {
			res := &pb.SearchLaptopResponse{
				Laptop: laptop,
			}

			err := stream.Send(res)
			if err != nil {
				return err
			}

			log.Printf("Sent laptop with id %s", laptop.GetId())
			return nil
		},
	)

	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}

	return nil
}

func (server *LaptopServer) UploadImageService(stream pb.LaptopService_UploadImageServiceServer) error {
	req, err := stream.Recv()

	if err != nil {
		return status.Errorf(codes.Unknown, "cannot find laptop %v", err)
	}

	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("receive an upload-image request for laptop %s with image type %s", laptopID, imageType)

	laptop := server.laptopStore.Find(laptopID)

	if laptop == nil {
		return status.Errorf(codes.InvalidArgument, "cannot found laptop %s", laptopID)
	}

	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		err := contextError(stream.Context())

		if err != nil {
			return err
		}

		log.Print("wait to receive more data")

		req, err := stream.Recv()

		if err == io.EOF {
			log.Print("No more data")
			break
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err)
		}

		chunk := req.GetChunkData()
		size := len(chunk)

		log.Printf("receive chunk data size %d ", size)

		imageSize += size

		if imageSize > maxImageSize {
			return status.Errorf(codes.InvalidArgument, "image is too big %d > %d", imageSize, maxImageSize)
		}

		_, err = imageData.Write(chunk)

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot write data: %v", err)
		}
	}

	imageID, err := server.imageStore.Save(laptopID, imageType, imageData)

	if err != nil {
		return status.Errorf(codes.Internal, "cannot store image %v", err)
	}

	res := &pb.UploadImageResponse{
		Id:   imageID,
		Size: uint32(imageSize),
	}

	err = stream.SendAndClose(res)

	if err != nil {
		return status.Errorf(codes.Unknown, "cannot send response: %v ", err)
	}

	log.Printf("save image with id: %s with size: %d", imageID, imageSize)

	return nil
}

func (server *LaptopServer) RateLaptopService(stream pb.LaptopService_RateLaptopServiceServer) error {
	for {
		err := contextError(stream.Context())

		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot read stream req: %v", err)
		}

		id := req.GetLaptopId()
		score := req.GetScore()

		log.Printf("received a rate-laptop request: id = %s, score = %.2f", id, score)

		laptop := server.laptopStore.Find(id)

		if laptop == nil {
			return status.Errorf(codes.NotFound, "cannot find laptop: %v", err)
		}

		rating, err := server.ratingStore.Add(id, score)

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot add rating: %v", err)
		}

		res := &pb.RateLaptopResponse{
			LaptopId:     id,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send res: %v", err)
		}
	}

	return nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Errorf(codes.Canceled, "request is canceled")
	case context.DeadlineExceeded:
		return status.Errorf(codes.DeadlineExceeded, "context deadline exceeded")
	default:
		return nil
	}
}
