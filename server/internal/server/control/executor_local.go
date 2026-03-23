package control

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LocalExecutor 本地命令执行器（单机模式）
type LocalExecutor struct{}

// NewLocalExecutor 创建本地执行器
func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

func (e *LocalExecutor) Execute(ctx context.Context, command string) ExecResult {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
		cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "LANGUAGE=C")
	}

	out, err := cmd.CombinedOutput()
	return ExecResult{Output: string(out), Err: err}
}

func (e *LocalExecutor) UploadFile(_ context.Context, remotePath string, reader io.Reader) error {
	dir := filepath.Dir(remotePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(remotePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return err
}

func (e *LocalExecutor) Close() error {
	return nil
}
