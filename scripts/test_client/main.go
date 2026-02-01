package main

import (
	"context"
	"log"
	"time"

	pb "github.com/AkikoAkaki/async-task-platform/api/proto"
	"github.com/AkikoAkaki/async-task-platform/internal/storage/redis"
)

func main() {
	ctx := context.Background()
	store := redis.NewStore("localhost:6379")

	// 创建一个任务
	task := &pb.Task{
		Id:          "test-dlq-001",
		Topic:       "test-topic",
		Payload:     `{"test":"dlq"}`,
		ExecuteTime: time.Now().Unix(),
		RetryCount:  0,
		MaxRetries:  1, // 设置为 1，方便测试一次失败就进死信队列
		CreatedAt:   time.Now().Unix(),
	}

	// 1. 入队
	log.Println("1. 入队任务...")
	if err := store.Add(ctx, task); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}

	// 2. 抓取
	log.Println("2. 抓取任务...")
	tasks, err := store.FetchAndHold(ctx, "test-topic", 10)
	if err != nil {
		log.Fatalf("FetchAndHold 失败: %v", err)
	}
	if len(tasks) == 0 {
		log.Fatal("未抓取到任务")
	}

	// 3. Nack (第一次失败)
	log.Println("3. Nack (第一次)...")
	if err := store.Nack(ctx, tasks[0]); err != nil {
		log.Fatalf("Nack 失败: %v", err)
	}

	// 4. 再次抓取
	log.Println("4. 再次抓取...")
	time.Sleep(100 * time.Millisecond)
	tasks, err = store.FetchAndHold(ctx, "test-topic", 10)
	if err != nil {
		log.Fatalf("FetchAndHold 失败: %v", err)
	}

	// 5. Nack (第二次失败，进 DLQ)
	log.Println("5. Nack (第二次，进 DLQ)...")
	if err := store.Nack(ctx, tasks[0]); err != nil {
		log.Fatalf("Nack 失败: %v", err)
	}

	// 6. 验证 DLQ
	log.Println("6. 验证 DLQ...")
	res, err := store.GetClient().LRange(ctx, "ddq:dlq", 0, -1).Result()
	if err != nil {
		log.Fatalf("LRange 失败: %v", err)
	}
	log.Printf("DLQ 中的任务数: %d", len(res))
	if len(res) > 0 {
		log.Printf("DLQ 内容: %s", res[0])
	}

	log.Println(" 测试完成")
}
