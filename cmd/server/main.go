// Package main 项目启动入口。
// 职责：负责配置加载、基础设施（Redis）初始化、gRPC 服务注册以及生命周期管理（优雅退出）。
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/AkikoAkaki/async-task-platfrom/api/proto"
	"github.com/AkikoAkaki/async-task-platfrom/internal/conf"
	"github.com/AkikoAkaki/async-task-platfrom/internal/queue"
	"github.com/AkikoAkaki/async-task-platfrom/internal/scheduler"
	"github.com/AkikoAkaki/async-task-platfrom/internal/storage/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 配置加载。
	// @Step: 依次尝试从当前目录及上级目录搜索 config 配置文件并加载。
	cfg, err := conf.Load("./config")
	if err != nil {
		cfg, err = conf.Load("../../config")
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
	}

	log.Printf("Starting %s [%s]...", cfg.App.Name, cfg.App.Env)

	// 2. 核心存储层初始化。
	// @Note: 使用 Redis 作为主存储，内部包含 JobStore 接口实现。
	store := redis.NewStore(cfg.Redis.Addr)

	// 3. 异步调度组件启动。
	// @Watchdog: 负责可见性超时任务的自动恢复。
	wd := scheduler.NewWatchdog(cfg.Queue, store)
    wd.Start()

	// 4. 网络层监听。
	// @Address: 默认从配置中读取 gRPC 端口号。
	addr := fmt.Sprintf(":%d", cfg.Server.GrpcPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 5. gRPC 服务注册。
	// @Services: 注册延迟队列业务服务，并开启反射（Reflection）以便于调试。
	s := grpc.NewServer()
	svc := queue.NewService(store)
	pb.RegisterDelayQueueServiceServer(s, svc)
	reflection.Register(s)

	// 6. 协议服务启动。
	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 7. 优雅关闭响应。
	// @Mechanism: 捕捉系统终止信号，确保清理后台协程并停止接受新连接后再退出。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	wd.Stop()
	s.GracefulStop()
	log.Println("Server stopped")
}
