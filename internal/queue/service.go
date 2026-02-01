// Package queue 实现了延迟队列的核心业务逻辑。
// 适用场景：处理 gRPC 请求，参数校验，并将任务调度下发至存储层。
package queue

import (
	"context"
	"time"

	pb "github.com/AkikoAkaki/async-task-platfrom/api/proto"
	"github.com/AkikoAkaki/async-task-platfrom/internal/common/errno"
	"github.com/AkikoAkaki/async-task-platfrom/internal/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service 延迟队列 gRPC 服务实现。
// @Description 充当业务网关（Gateway），负责输入校验、ID 生成、任务规整，最后通过 JobStore 接口实现持久化。
type Service struct {
	pb.UnimplementedDelayQueueServiceServer
	store storage.JobStore // 任务持久化后端实现
}

// NewService 创建延迟队列服务实例。
// @Param store: 任务存取引擎的实现，通常为 Redis 实现。
func NewService(store storage.JobStore) *Service {
	return &Service{
		store: store,
	}
}

// Enqueue 处理任务提交请求（入队）。
// @Complexity: O(log(N))，取决于存储实现。
// @Return: 成功则返回任务分配的唯一 ID；失败则返回 gRPC 错误码。
func (s *Service) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	// 1. 参数校验。
	// @Validation: 检查 Topic、Payload 是否为空，延时时间是否合法。
	if req.Topic == "" || req.Payload == "" {
		return nil, status.Error(codes.InvalidArgument, errno.ErrInvalidParam.Message)
	}
	if req.DelaySeconds < 0 {
		return nil, status.Error(codes.InvalidArgument, "delay_seconds must be >= 0")
	}

	// 2. 身份标识分配。
	// @Note: 优先使用客户端传入的 ID 以支持幂等提交，否则由系统自动生成 UUID。
	taskID := req.Id
	if taskID == "" {
		taskID = uuid.New().String()
	}

	// 3. 策略初始化。
	// @Default: 若未指定最大重试次数，则赋予系统预设默认值（3次）。
	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3 
	}

	// 4. 构造任务实体快照。
	task := &pb.Task{
		Id:          taskID,
		Topic:       req.Topic,
		Payload:     req.Payload,
		ExecuteTime: time.Now().Add(time.Duration(req.DelaySeconds) * time.Second).Unix(),
		RetryCount:  0,
		MaxRetries:  maxRetries,
		CreatedAt:   time.Now().Unix(),
	}

	// 5. 调用持久化层。
	// @ErrorHandling: 若存储层故障（如 Redis 连接断开），返回 Internal 错误给客户端以便重试。
	if err := s.store.Add(ctx, task); err != nil {
		return &pb.EnqueueResponse{
			Success:      false,
			ErrorMessage: "failed to store task",
		}, status.Error(codes.Internal, err.Error())
	}

	return &pb.EnqueueResponse{
		Success: true,
		Id:      taskID,
	}, nil
}

// Retrieve 拉取任务（Admin/Monitoring 扩展接口）。
// @Status: Unimplemented
func (s *Service) Retrieve(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Retrieve not implemented yet")
}

// Delete 撤销任务。
// @Status: Unimplemented
func (s *Service) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented yet")
}
