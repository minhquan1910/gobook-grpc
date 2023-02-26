package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"gobook/pb"
	"gobook/sample"
	"gobook/service"
	"gobook/util"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

func accessibleRoles() map[string][]string {
	const laptopServicePath = "/pb.LaptopService/"

	return map[string][]string{
		laptopServicePath + "CreateLaptopService": {"admin"},
		laptopServicePath + "UploadImageService":  {"admin"},
		laptopServicePath + "RateLaptopService":   {"admin", "user"},
	}
}

const (
	username         = "quan"
	password         = "secret"
	serverCertFile   = "cert/server-cert.pem"
	serverKeyFile    = "cert/server-key.pem"
	clientCACertFile = "cert/ca-cert.pem"
)

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := os.ReadFile(clientCACertFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(config), nil
}

func main() {

	config, err := util.LoadConfig("./app.env")
	if err != nil {
		log.Fatal("cannot load .env file ", err)
	}
	port := flag.Int("port", 0, "server port running")
	flag.Parse()

	log.Printf("server running on port %d ", *port)

	//Interceptor
	jwtManager := service.NewJWTToken(config.TokenSymmetricKey)
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())
	tlsCreadential, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load tls", err)
	}
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
		grpc.Creds(tlsCreadential),
	}

	//Create store
	store := service.NewInMemoryLaptopStore()
	laptop := sample.NewLaptop()
	log.Println("Laptop ID: ", laptop.Id)
	err = store.Save(laptop)
	if err != nil {
		log.Fatal("cannot save laptop ", err)
	}
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()
	//Create Server
	laptopServer := service.NewLaptopServer(store, imageStore, ratingStore)
	//Create grpc server
	grpcServer := grpc.NewServer(serverOptions...)
	//RegisterServer
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	//Create Auth Server
	//Create User
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("cannot hash password ", err)
	}
	user := &service.User{
		Username:      username,
		HasedPassword: string(hashedPassword),
		Role:          "admin",
	}
	userStore := service.NewInMemoryUserStore()
	err = userStore.Save(user)
	if err != nil {
		log.Fatal("cannot save user ", err)
	}
	authServer := service.NewAuthServer(userStore, jwtManager)
	pb.RegisterAuthServiceServer(grpcServer, authServer)
	reflection.Register(grpcServer)
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
