package main

import (
	"context"
	"log"
	"time"

	pb "github.com/AkikoAkaki/async-task-platform/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 连接到 gRPC 服务器
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	client := pb.NewDelayQueueServiceClient(conn)

	// 测试 1: 提交一个 5 秒延迟的任务
	log.Println("=== Test 1: Enqueue task with 5s delay ===")
	resp1, err := client.Enqueue(context.Background(), &pb.EnqueueRequest{
		Topic:        "order-timeout",
		Payload:      `{"order_id": 12345, "user_id": 67890}`,
		DelaySeconds: 5,
	})
	if err != nil {
		log.Fatalf("Enqueue failed: %v", err)
	}
	log.Printf("✅ Task enqueued successfully! ID: %s", resp1.Id)

	// 测试 2: 提交一个立即执行的任务
	log.Println("\n=== Test 2: Enqueue task with 0s delay (immediate) ===")
	resp2, err := client.Enqueue(context.Background(), &pb.EnqueueRequest{
		Topic:        "notification",
		Payload:      `{"message": "Hello from async-task-platform!", "type": "email"}`,
		DelaySeconds: 0,
	})
	if err != nil {
		log.Fatalf("Enqueue failed: %v", err)
	}
	log.Printf("✅ Task enqueued successfully! ID: %s", resp2.Id)

	// 测试 3: 提交多个任务
	log.Println("\n=== Test 3: Enqueue multiple tasks ===")
	for i := 1; i <= 3; i++ {
		resp, err := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Topic:        "batch-job",
			Payload:      `{"job_id": ` + string(rune(i+'0')) + `}`,
			DelaySeconds: int64(i * 2),
		})
		if err != nil {
			log.Printf("❌ Task %d failed: %v", i, err)
			continue
		}
		log.Printf("✅ Task %d enqueued! ID: %s, Delay: %ds", i, resp.Id, i*2)
	}

	log.Println("\n=== All tasks submitted! ===")
	log.Println("Check the Worker logs to see task execution...")
	log.Println("Waiting 15 seconds for tasks to complete...")
	time.Sleep(15 * time.Second)
	log.Println("✅ Test completed!")
}
