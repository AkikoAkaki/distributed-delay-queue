package queue

import (
	"context"
	"testing"

	pb "github.com/AkikoAkaki/async-task-platform/api/proto"
	"github.com/AkikoAkaki/async-task-platform/internal/storage/mocks"
	"go.uber.org/mock/gomock"
)

func TestEnqueue(t *testing.T) {
	// 1. 初始化 Controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 2. 创建 Mock 对象
	mockStore := mocks.NewMockJobStore(ctrl)
	svc := NewService(mockStore)

	// 3. 定义测试用例
	tests := []struct {
		name    string
		req     *pb.EnqueueRequest
		mock    func()
		wantErr bool
	}{
		{
			name: "Success",
			req: &pb.EnqueueRequest{
				Topic:        "test",
				Payload:      "{}",
				DelaySeconds: 10,
			},
			mock: func() {
				// 预期 Store.Add 会被调用一次，且返回 nil (无错误)
				mockStore.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Invalid Param",
			req: &pb.EnqueueRequest{
				Topic: "", // Invalid
			},
			mock: func() {
				// 预期 Store.Add 不会被调用
			},
			wantErr: true,
		},
	}

	// 4. 执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := svc.Enqueue(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Enqueue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
