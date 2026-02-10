package worker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"server/internal/buildworker/queue"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/buildqueue"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

type Executor struct {
	sshClient *utils.SSHClient
	queue     *queue.BuildQueue
	taskID    int
}

func NewExecutor(sshClient *utils.SSHClient, q *queue.BuildQueue, taskID int) *Executor {
	return &Executor{
		sshClient: sshClient,
		queue:     q,
		taskID:    taskID,
	}
}

// ExecuteWithProgress 执行命令并解析进度
func (e *Executor) ExecuteWithProgress(cmd string, timeout time.Duration) (string, error) {
	// 创建 SSH 会话
	if e.sshClient.Client == nil {
		return "", fmt.Errorf("SSH client not connected")
	}

	session, err := e.sshClient.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	// 获取输出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return "", err
	}

	// 启动命令
	if err := session.Start(cmd); err != nil {
		return "", fmt.Errorf("启动命令失败: %v", err)
	}

	// 读取输出
	var outputBuilder strings.Builder
	var logLines []string

	// 进度解析正则
	progressRegex := regexp.MustCompile(`\[PROGRESS[=:]\s*(\d+)%?\]`)
	stepRegex := regexp.MustCompile(`\[(INFO|SUCCESS|WARN|ERROR|STEP)\]\s*(.+)`)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		// 读取 stdout
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")

			// 保留最近100行日志
			logLines = append(logLines, line)
			if len(logLines) > 100 {
				logLines = logLines[1:]
			}

			// 检查是否取消
			if e.queue.IsCancelled(e.taskID) {
				logx.Infof("Task %d cancelled, sending signal", e.taskID)
				session.Signal(ssh.SIGINT)
				done <- fmt.Errorf("任务已取消")
				return
			}

			// 解析进度
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
				progress, _ := strconv.Atoi(matches[1])
				e.updateProgress(progress, "", logLines)
			}

			// 解析步骤
			if matches := stepRegex.FindStringSubmatch(line); len(matches) > 2 {
				step := matches[2]
				e.updateProgress(0, step, logLines) // 0 表示不更新进度
			}
		}

		// 读取 stderr
		stderrBytes, _ := io.ReadAll(stderr)
		if len(stderrBytes) > 0 {
			stderrStr := string(stderrBytes)
			outputBuilder.WriteString("=== STDERR ===\n")
			outputBuilder.WriteString(stderrStr)
			logLines = append(logLines, "STDERR: "+stderrStr)
		}

		done <- session.Wait()
	}()

	// 等待完成或超时
	select {
	case err := <-done:
		return outputBuilder.String(), err
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return outputBuilder.String(), fmt.Errorf("构建超时 (%v)", timeout)
	}
}

func (e *Executor) updateProgress(progress int, step string, logLines []string) {
	current, _ := e.queue.GetProgress(e.taskID)
	if current == nil {
		current = &buildqueue.BuildProgress{
			TaskID: e.taskID,
			Status: model.BuildStatusBuilding,
		}
	}

	// 只有新进度更大时才更新
	if progress > 0 && progress > current.Progress {
		current.Progress = progress
	}

	if step != "" {
		current.CurrentStep = step
	}

	current.LogLines = logLines
	e.queue.UpdateProgress(current)
}
