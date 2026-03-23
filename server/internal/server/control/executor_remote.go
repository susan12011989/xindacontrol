package control

import (
	"context"
	"io"
	"server/internal/server/utils"
	"time"
)

// SSH 命令默认超时
const defaultSSHCommandTimeout = 30 * time.Second

// RemoteExecutor 远程命令执行器（多机模式，基于 SSH）
type RemoteExecutor struct {
	client *utils.PooledSSHClient
}

// NewRemoteExecutor 通过已有 SSH 连接池客户端创建远程执行器
func NewRemoteExecutor(client *utils.PooledSSHClient) *RemoteExecutor {
	return &RemoteExecutor{client: client}
}

func (e *RemoteExecutor) Execute(ctx context.Context, command string) ExecResult {
	// 从 context 取超时，没有则用默认 30 秒
	timeout := defaultSSHCommandTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	output, err := e.client.ExecuteCommandWithTimeout(command, timeout)
	return ExecResult{Output: output, Err: err}
}

func (e *RemoteExecutor) UploadFile(_ context.Context, remotePath string, reader io.Reader) error {
	return e.client.UploadFile(remotePath, reader)
}

func (e *RemoteExecutor) Close() error {
	// 不关闭连接池中的客户端，由连接池统一管理生命周期
	return nil
}
