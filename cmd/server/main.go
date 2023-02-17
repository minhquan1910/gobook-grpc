package main

import (
	"flag"
	"fmt"
	"gobook/pb"
	"gobook/service"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 0, "server port running")
	flag.Parse()

	log.Printf("server running on port %d ", *port)

	//Create store
	store := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	//Create Server
	laptopServer := service.NewLaptopServer(store, imageStore, nil)
	//Create grpc server
	grpcServer := grpc.NewServer()
	//RegisterServer
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	address := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("cannot listen to port : ", err)
	}

	err = grpcServer.Serve(listener)

	if err != nil {
		log.Fatal("cannot start server : ", err)
	}
}
