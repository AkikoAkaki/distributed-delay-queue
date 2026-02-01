package main

import (
	"context"
	"log"

	pb "github.com/AkikoAkaki/async-task-platfrom/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDelayQueueServiceClient(conn)

	log.Println("Sending task (delay 5s)...")
	_, err = c.Enqueue(context.Background(), &pb.EnqueueRequest{
		Topic:        "email",
		Payload:      `{"user": "alice"}`,
		DelaySeconds: 5,
		MaxRetries:   1, // 设置为 1，方便测试一次失败就进死信
	})
	if err != nil {
		log.Fatalf("Enqueue failed: %v", err)
	}
	log.Println("Task sent!")
}
