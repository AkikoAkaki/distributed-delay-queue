package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AkikoAkaki/async-task-platform/internal/conf"
	"github.com/AkikoAkaki/async-task-platform/internal/storage/redis"
)

func main() {
	// 1. 加载配置
	cfg, err := conf.Load("./config")
	if err != nil {
		cfg, err = conf.Load("../../config")
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}

	// 2. 使用配置连接 Redis
	store := redis.NewStore(cfg.Redis.Addr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Worker started, polling for tasks...")

	// 使用 WaitGroup 保证退出时处理完当前 Loop
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second) // 轮询间隔 1秒
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// 收到停止信号，退出循环
				return
			case <-ticker.C:
				// 1. 拉取任务
				tasks, err := store.FetchAndHold(ctx, "default", 10)
				if err != nil {
					log.Printf("Error polling tasks: %v", err)
					continue
				}

				// 2. 执行任务 (MVP: 仅打印)
				if len(tasks) > 0 {
					log.Printf("--- Processed %d tasks ---", len(tasks))
					for _, t := range tasks {
						// 工业级：这里应该扔给一个 Worker Pool 线程池去并发执行，而不是串行阻塞
						log.Printf("[EXECUTE] TaskID: %s, Payload: %s, Delay: %ds",
							t.Id, t.Payload, time.Now().Unix()-t.ExecuteTime)

						// 3. 任务执行成功后，调用 Ack 确认完成
						// @Critical: 如果不调用 Ack，任务会永远停留在 Running 状态，
						// 最终被 Watchdog 认为超时并重新入队，导致重复执行。
						if err := store.Ack(ctx, t.Id); err != nil {
							log.Printf("[ERROR] Ack failed for task %s: %v", t.Id, err)
							// 注意：Ack 失败意味着任务状态不一致，Watchdog 会恢复它
						} else {
							log.Printf("[ACK] Task %s completed successfully", t.Id)
						}
					}
				}
			}
		}
	}()

	// 优雅退出监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Worker shutting down...")
	cancel()  // 通知 Loop 停止
	wg.Wait() // 等待 Loop 彻底结束
	log.Println("Worker stopped")
}
