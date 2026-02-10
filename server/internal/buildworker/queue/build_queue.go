package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"server/pkg/buildqueue"
	"time"

	"github.com/redis/go-redis/v9"
)

type BuildQueue struct {
	rds *redis.Client
	ctx context.Context
}

func NewBuildQueue(rds *redis.Client) *BuildQueue {
	return &BuildQueue{
		rds: rds,
		ctx: context.Background(),
	}
}

// Enqueue 任务入队
func (q *BuildQueue) Enqueue(task *buildqueue.BuildTaskMessage) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}

	return q.rds.RPush(q.ctx, buildqueue.BuildTaskQueueKey, data).Err()
}

// Dequeue 从队列获取任务（阻塞）
func (q *BuildQueue) Dequeue(timeout time.Duration) (*buildqueue.BuildTaskMessage, error) {
	result, err := q.rds.BLPop(q.ctx, timeout, buildqueue.BuildTaskQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 超时，无任务
		}
		return nil, err
	}

	if len(result) < 2 {
		return nil, nil
	}

	var task buildqueue.BuildTaskMessage
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("解析任务失败: %w", err)
	}

	return &task, nil
}

// UpdateProgress 更新任务进度
func (q *BuildQueue) UpdateProgress(progress *buildqueue.BuildProgress) error {
	progress.UpdatedAt = time.Now()
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%d", progress.TaskID)
	return q.rds.HSet(q.ctx, buildqueue.BuildTaskProgressKey, key, data).Err()
}

// GetProgress 获取任务进度
func (q *BuildQueue) GetProgress(taskID int) (*buildqueue.BuildProgress, error) {
	key := fmt.Sprintf("%d", taskID)
	data, err := q.rds.HGet(q.ctx, buildqueue.BuildTaskProgressKey, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var progress buildqueue.BuildProgress
	if err := json.Unmarshal([]byte(data), &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

// MarkCancelled 标记任务为已取消
func (q *BuildQueue) MarkCancelled(taskID int) error {
	return q.rds.SAdd(q.ctx, buildqueue.BuildTaskCancelKey, taskID).Err()
}

// IsCancelled 检查任务是否已取消
func (q *BuildQueue) IsCancelled(taskID int) bool {
	result, err := q.rds.SIsMember(q.ctx, buildqueue.BuildTaskCancelKey, taskID).Result()
	if err != nil {
		return false
	}
	return result
}

// RemoveCancelMark 移除取消标记
func (q *BuildQueue) RemoveCancelMark(taskID int) error {
	return q.rds.SRem(q.ctx, buildqueue.BuildTaskCancelKey, taskID).Err()
}

// ClearProgress 清除任务进度
func (q *BuildQueue) ClearProgress(taskID int) error {
	key := fmt.Sprintf("%d", taskID)
	return q.rds.HDel(q.ctx, buildqueue.BuildTaskProgressKey, key).Err()
}
