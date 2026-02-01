package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	pb "github.com/AkikoAkaki/async-task-platform/api/proto"
	"github.com/AkikoAkaki/async-task-platform/internal/storage/redis"
)

func main() {
	ctx := context.Background()
	store := redis.NewStore("localhost:6379")

	// 清空测试数据
	store.GetClient().Del(ctx, "ddq:tasks", "ddq:running", "ddq:dlq")

	// --- 任务配置 ---
	task := &pb.Task{
		Id:          "test-task-001",
		Topic:       "test-topic",
		Payload:     `{"action":"test"}`,
		ExecuteTime: time.Now().Unix(),
		RetryCount:  0,
		MaxRetries:  1, // 设置为 1，第二次 Nack 就会进死信队列
		CreatedAt:   time.Now().Unix(),
	}

	// 1. 入队
	printHeader("阶段 1: 入队任务")
	if err := store.Add(ctx, task); err != nil {
		log.Fatalf("Add 失败: %v", err)
	}
	fmt.Printf(" 任务已入队: %s\n", task.Id)

	// 2. 第一次抓取
	printHeader("阶段 2: Worker 第一次抓取")
	tasks, err := store.FetchAndHold(ctx, "test-topic", 10)
	if err != nil {
		log.Fatalf("FetchAndHold 失败: %v", err)
	}
	if len(tasks) == 0 {
		log.Fatal("未抓取到任务")
	}
	t1 := tasks[0]
	fmt.Printf(" 抓取到任务: %s, RetryCount=%d\n", t1.Id, t1.RetryCount)

	// 3. 第一次失败
	printHeader("阶段 3: 模拟第一次失败 (Nack)")
	if err := store.Nack(ctx, t1); err != nil {
		log.Fatalf("Nack 失败: %v", err)
	}
	fmt.Printf(" Nack 成功，任务应重新入队\n")

	// 4. 第二次抓取 (重试)
	printHeader("阶段 4: Worker 第二次抓取 (重试)")
	time.Sleep(100 * time.Millisecond) // 等待重新入队
	tasks, err = store.FetchAndHold(ctx, "test-topic", 10)
	if err != nil {
		log.Fatalf("FetchAndHold 失败: %v", err)
	}
	if len(tasks) == 0 {
		log.Fatal("未抓取到重试任务")
	}
	t2 := tasks[0]
	fmt.Printf(" 抓取到重试任务: %s, RetryCount=%d\n", t2.Id, t2.RetryCount)

	// 5. 第二次失败 (最终失败)
	printHeader("阶段 5: 模拟第二次失败 (进入 DLQ)")
	if err := store.Nack(ctx, t2); err != nil {
		log.Fatalf("Nack 失败: %v", err)
	}
	fmt.Printf(" Nack 成功，任务应进入死信队列\n")

	// 6. 验证结果
	printHeader("阶段 6: 验证死信队列")
	res, err := store.GetClient().LRange(ctx, "ddq:dlq", 0, -1).Result()
	if err != nil {
		log.Fatalf("LRange 失败: %v", err)
	}
	if len(res) == 0 {
		log.Fatal("验证失败: DLQ 为空")
	}

	var dlqTask pb.Task
	if err := json.Unmarshal([]byte(res[0]), &dlqTask); err != nil {
		log.Fatalf("Unmarshal 失败: %v", err)
	}
	fmt.Printf(" DLQ 中找到任务: %s, RetryCount=%d\n", dlqTask.Id, dlqTask.RetryCount)

	fmt.Println("\n 全部测试通过！任务生命周期验证成功")
}

func printHeader(title string) {
	fmt.Printf("\n========== %s ==========\n", title)
}
