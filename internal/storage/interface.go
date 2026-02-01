// Package storage 定义了系统任务持久化层的抽象规范。
// 职责:屏蔽底层存储介质(如 Redis、MySQL、RocksDB)的差异,
// 为业务逻辑层提供统一的任务 CRUD 接口,支持分布式环境下的任务状态管理。
//
//go:generate mockgen -source=interface.go -destination=mocks/store_mock.go -package=mocks
package storage

import (
	"context"

	pb "github.com/AkikoAkaki/async-task-platform/api/proto"
)

// JobStore 定义了任务存储层的行为契约。
// @Description 实现类必须保证操作的原子性(尤其是 GetReady 中的"拉取并隐藏/移除"逻辑),
// 并负责处理底层驱动的连接池管理及重试机制。
type JobStore interface {
	// Add 将任务持久化至存储引擎。
	// @Param ctx: 传递上下文链路信息,支持超时取消。
	// @Param task: 待存储的任务原始数据,调用方需确保 task 字段合法。
	// @Return: 存储失败时返回包含具体原因的 errno,如存储连接异常或数据校验失败。
	Add(ctx context.Context, task *pb.Task) error

	// FetchAndHold 批量获取并锁定已到执行时间的任务列表。
	// @Description 该方法通常包含"读取-修改"的复合操作,实现者需确保在并发环境下不重复下发同一任务。
	// @Param topic: 任务所属的业务主题分类。
	// @Param limit: 本次拉取任务的最大数量上限,用于防止内存溢出。
	// @Return: 返回待处理的任务切片;若当前无到期任务,返回空切片及 nil error。
	FetchAndHold(ctx context.Context, topic string, limit int64) ([]*pb.Task, error)

	// Remove 根据任务唯一标识从存储中彻底删除任务。
	// @Description 常用于任务撤回或任务成功处理后的终结操作。
	// @Param id: 任务全局唯一 ID。
	// @Return: 若 ID 不存在,实现者应根据业务需求决定是否返回特定错误。
	Remove(ctx context.Context, id string) error

	Ack(ctx context.Context, id string) error

	Nack(ctx context.Context, task *pb.Task) error

	CheckAndMoveExpired(ctx context.Context, visibilityTimeout int64, maxRetries int32) error
}
