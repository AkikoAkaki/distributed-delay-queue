// Package scheduler 负责队列的后台调度任务。
// 核心逻辑：包含 Watchdog（看门狗）机制，用于监控和恢复可见性超时或执行失败的任务。
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/AkikoAkaki/distributed-delay-queue/internal/conf"
	"github.com/AkikoAkaki/distributed-delay-queue/internal/storage"
)

// Watchdog 看门狗组件，负责定期扫描并恢复异常任务（如 Worker 宕机导致的未 Ack 任务）。
// @ThreadSafe: 内部状态受协程生命周期管理，支持跨协程安全启动/停止。
type Watchdog struct {
	store    storage.JobStore // 任务持久化存储接口
	interval time.Duration    // 扫描频率
	timeout  int64            // 可见性超时阈值（秒）
	maxRetry int32            // 任务最大重试次数
	
	quit     chan struct{}    // 退出信号通道
	wg       sync.WaitGroup   // 等待协程关闭
}

// NewWatchdog 根据配置初始化 Watchdog 实例。
// @Param cfg: 队列全局配置，包含扫描间隔、超时时间等。
// @Param store: 实现 JobStore 接口的任务存储器。
func NewWatchdog(cfg conf.QueueConfig, store storage.JobStore) *Watchdog {
	return &Watchdog{
		store:    store,
		interval: time.Duration(cfg.WatchdogInterval) * time.Second,
		timeout:  int64(cfg.VisibilityTimeout),
		maxRetry: int32(cfg.MaxRetries),
		quit:     make(chan struct{}),
	}
}

// Start 异步启动看门狗循环。
func (w *Watchdog) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		log.Printf("Watchdog started. Interval: %v, Timeout: %ds, MaxRetries: %d", w.interval, w.timeout, w.maxRetry)

		for {
			select {
			case <-w.quit:
				return
			case <-ticker.C:
				// 定期触发异常任务恢复
				w.recover()
			}
		}
	}()
}

// Stop 停止看门狗循环并等待协程安全退出。
func (w *Watchdog) Stop() {
	close(w.quit)
	w.wg.Wait()
	log.Println("Watchdog stopped")
}

// recover 执行任务恢复逻辑。
// @Algorithm: 调用存储层的 CheckAndMoveExpired，利用 Lua 脚本保证“检测超时+重入队”的原子性。
// @Note: 恢复过程带有重试次数限制，超过限制的任务将进入死信队列（DLQ）。
func (w *Watchdog) recover() {
	// 设置单次恢复任务的 Context 超时，防止因存储层压力过大导致协程堆积。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.store.CheckAndMoveExpired(ctx, w.timeout, w.maxRetry); err != nil {
		log.Printf("Watchdog recover error: %v", err)
	}
}
