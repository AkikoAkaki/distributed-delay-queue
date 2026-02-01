package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/AkikoAkaki/async-task-platfrom/api/proto"
	"github.com/AkikoAkaki/async-task-platfrom/internal/storage/redis"
)

func main() {
	ctx := context.Background()
	store := redis.NewStore("localhost:6379")

	printHeader("环境清理")
	store.GetClient().Del(ctx, "ddq:tasks", "ddq:running", "ddq:dlq")
	fmt.Println("✅ 已清空 Redis 旧数据 (tasks, running, dlq)")

	// --- 任务配置 ---
	taskID := "test-task-001"
	task := &pb.Task{
		Id:          taskID,
		Topic:       "test-topic",
		Payload:     "hello world",
		MaxRetries:  2,
		ExecuteTime: time.Now().Unix(),
	}

	// 1. 入队
	printHeader("阶段 1: 任务初始入队 (Add)")
	if err := store.Add(ctx, task); err != nil {
		log.Fatalf("❌ Add 失败: %v", err)
	}
	fmt.Printf("成功将任务 [%s] 加入 PENDING 队列\n", taskID)

	// 2. 第一次抓取
	printHeader("阶段 2: 第一次消费 (FetchAndHold)")
	tasks, _ := store.FetchAndHold(ctx, "test-topic", 1)
	if len(tasks) == 0 {
		log.Fatal("❌ 未能获取到任务")
	}
	t1 := tasks[0]
	fmt.Printf("📥 抓取成功 | ID: %s | RetryCount: %d\n", t1.Id, t1.RetryCount)

	// 3. 第一次失败
	printHeader("阶段 3: 模拟第一次失败 (Nack)")
	if err := store.Nack(ctx, t1); err != nil {
		log.Fatalf("❌ Nack 失败: %v", err)
	}
	fmt.Println("🔄 调用 Nack: 任务应因 RetryCount < MaxRetries 而回到 PENDING")

	// 4. 第二次抓取 (重试)
	printHeader("阶段 4: 第二次消费 (重试抓取)")
	tasks, _ = store.FetchAndHold(ctx, "test-topic", 1)
	if len(tasks) == 0 {
		log.Fatal("❌ 重试任务未能重新入队")
	}
	t2 := tasks[0]
	fmt.Printf("📥 重新抓取 | ID: %s | RetryCount: %d\n", t2.Id, t2.RetryCount)

	// 5. 第二次失败 (最终失败)
	printHeader("阶段 5: 模拟第二次失败 (Nack -> DLQ)")
	if err := store.Nack(ctx, t2); err != nil {
		log.Fatalf("❌ Nack 失败: %v", err)
	}
	fmt.Println("💀 调用 Nack: 任务应因达到 MaxRetries 进入 DLQ")

	// 6. 验证结果
	printHeader("阶段 6: 最终状态校验")
	res, _ := store.GetClient().LRange(ctx, "ddq:dlq", 0, -1).Result()
	if len(res) > 0 {
		fmt.Printf("🏆 验证成功！死信队列 (DLQ) 捕获到目标任务:\n")
		for _, item := range res {
			fmt.Printf("   📝 内容: %s\n", item)
		}
	} else {
		fmt.Println("❌ 验证失败: DLQ 为空")
	}

	fmt.Println("\n" + "========================================")
	fmt.Println("  🎉 分布式延迟队列全生命周期测试完成")
	fmt.Println("========================================")
}

func printHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}
