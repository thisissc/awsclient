package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	msg "github.com/thisissc/awsclient/resolver/example/lib"
	"google.golang.org/grpc"
)

const (
	MaxPort = 8089
	MinPort = 8080
)

type Hello struct{}

func (h *Hello) HelloServer(ctx context.Context, in *msg.HelloRequest) (*msg.HelloResponse, error) {
	log.Println(in)
	return &msg.HelloResponse{Ok: true}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	randPort := r1.Intn(MaxPort-MinPort) + MinPort
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", randPort)
	log.Println(grpcAddr)

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Println(err)
	}

	grpcServer := grpc.NewServer()
	msg.RegisterHelloServiceServer(grpcServer, &Hello{})

	if err := grpcServer.Serve(l); err != nil {
		log.Println(err)
	}

}
