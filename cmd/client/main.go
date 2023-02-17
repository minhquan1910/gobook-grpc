package main

import (
	"flag"
	"gobook/client"
	"gobook/sample"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	serverAddress := flag.String("address", "", "server address")
	flag.Parse()

	log.Printf("client running on address: %s", *serverAddress)

	conn, err := grpc.Dial(*serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal("cannot dial server", err)
	}

	laptopClient := client.NewLaptopClient(conn)

	// for i := 0; i < 10; i++ {
	// 	laptop := sample.NewLaptop()
	// 	laptopClient.CreateLaptop(laptop)
	// }

	// filter := &pb.Filter{
	// 	MaxPriceUsd: 3000,
	// 	MinCpuCores: 4,
	// 	MinCpuGhz:   2.5,
	// 	MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	// }

	// laptopClient.SearchLaptop(filter)
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.Id, "tmp/laptop.jpg")
}
