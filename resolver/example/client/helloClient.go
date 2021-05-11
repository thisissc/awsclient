package main

import (
	"context"
	"log"
	"time"

	_ "github.com/thisissc/awsclient/resolver"
	msg "github.com/thisissc/awsclient/resolver/example/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	endPoint := "awsecssrv://api02/chat-ws"

	conn, err := grpc.Dial(endPoint, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	for {
		client := msg.NewHelloServiceClient(conn)

		resp, err := client.HelloServer(context.TODO(), &msg.HelloRequest{
			Text: "hello",
		})
		if err != nil {
			log.Println(err)
		}

		log.Println(resp)

		time.Sleep(3 * time.Second)
	}
}
