package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/ssh"
)

// WebSSHMessage WebSocket消息格式
type WebSSHMessage struct {
	Type string `json:"type"` // input/resize/ping/pong
	Data string `json:"data"` // 数据内容
	Rows int    `json:"rows"` // 终端行数
	Cols int    `json:"cols"` // 终端列数
}

// HandleWebSSH 处理WebSSH连接
func HandleWebSSH(ws *websocket.Conn, serverId int) error {
	// 获取SSH客户端
	pooledClient, err := GetSSHClient(serverId)
	if err != nil {
		return fmt.Errorf("获取SSH连接失败: %v", err)
	}

	// 判断是否本地服务器
	isLocal := false
	if pooledClient != nil && pooledClient.SSHClient != nil {
		h := strings.TrimSpace(strings.ToLower(pooledClient.SSHClient.Host))
		if h == "127.0.0.1" || h == "localhost" || h == "::1" {
			isLocal = true
		}
	}

	if isLocal || pooledClient == nil || pooledClient.Client == nil {
		// 本地模式：使用本地 shell + PTY
		return handleLocalWebSSH(ws)
	}

	// 创建SSH会话
	session, err := pooledClient.Client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // 启用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度
		ssh.TTY_OP_OSPEED: 14400, // 输出速度
	}

	// 请求伪终端
	if err := session.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		return fmt.Errorf("请求PTY失败: %v", err)
	}

	// 获取输入输出管道
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("获取stdin失败: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout失败: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr失败: %v", err)
	}

	// 启动shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("启动shell失败: %v", err)
	}

	// 使用context和WaitGroup协调goroutine退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保所有goroutine收到取消信号

	var wg sync.WaitGroup
	wg.Add(3)

	// goroutine 1: 读取SSH输出并发送到WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024) // 32KB缓冲区
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					logx.Errorf("读取stdout失败: %v", err)
				}
				cancel() // 通知其他goroutine退出
				return
			}
			if n > 0 {
				msg := map[string]interface{}{
					"type": "output",
					"data": string(buf[:n]),
				}
				// 设置写入超时，避免长时间阻塞
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(msg); err != nil {
					logx.Errorf("发送到WebSocket失败: %v", err)
					cancel() // 通知其他goroutine退出
					return
				}
			}
		}
	}()

	// goroutine 2: 读取SSH错误输出并发送到WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					logx.Errorf("读取stderr失败: %v", err)
				}
				cancel() // 通知其他goroutine退出
				return
			}
			if n > 0 {
				msg := map[string]interface{}{
					"type": "output",
					"data": string(buf[:n]),
				}
				// 设置写入超时
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(msg); err != nil {
					logx.Errorf("发送到WebSocket失败: %v", err)
					cancel() // 通知其他goroutine退出
					return
				}
			}
		}
	}()

	// goroutine 3: 读取WebSocket消息并发送到SSH
	go func() {
		defer wg.Done()
		defer stdin.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 设置读取超时
			ws.SetReadDeadline(time.Now().Add(60 * time.Second))
			var msg WebSSHMessage
			if err := ws.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					logx.Errorf("读取WebSocket失败: %v", err)
				}
				cancel() // 通知其他goroutine退出
				return
			}

			switch msg.Type {
			case "input":
				// 用户输入
				if _, err := stdin.Write([]byte(msg.Data)); err != nil {
					logx.Errorf("写入stdin失败: %v", err)
					cancel()
					return
				}
			case "resize":
				// 调整终端大小
				if msg.Rows > 0 && msg.Cols > 0 {
					session.WindowChange(msg.Rows, msg.Cols)
				}
			case "ping":
				// 心跳请求，返回pong
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(map[string]interface{}{"type": "pong"}); err != nil {
					logx.Errorf("发送pong失败: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// 等待会话结束
	session.Wait()
	wg.Wait()

	return nil
}

// handleLocalWebSSH 本地 WebShell（WS <-> 本地 Shell）
func handleLocalWebSSH(ws *websocket.Conn) error {
	// 选择本地 shell
	var shell string
	var args []string
	if runtime.GOOS == "windows" {
		// 优先 powershell；设置 UTF-8 编码与不退出
		shell = "powershell.exe"
		args = []string{
			"-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-NoExit",
			"-Command",
			"$OutputEncoding=[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); " +
				"[Console]::InputEncoding=[System.Text.UTF8Encoding]::new(); " +
				"chcp 65001 > $null",
		}
	} else {
		shell = "bash"
		args = []string{"-l"}
	}

	cmd := exec.Command(shell, args...)
	// 设置终端类型，提升兼容性
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 优先启动 PTY（跨平台支持，Windows 需新版系统支持 ConPTY）
	f, err := pty.Start(cmd)
	if err != nil {
		// Windows 本地体验要求强交互，直接返回错误引导用户切换到系统支持的环境
		return fmt.Errorf("本地终端不支持 PTY，请在支持 ConPTY 的环境运行或使用远程服务器")
	}

	// PTY 模式：单一流
	defer func() {
		_ = f.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// 使用context协调goroutine退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保所有goroutine收到取消信号

	var wg sync.WaitGroup
	wg.Add(2)

	// 输出 -> WS
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := f.Read(buf)
			if n > 0 {
				// 设置写入超时
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(map[string]interface{}{"type": "output", "data": string(buf[:n])}); err != nil {
					logx.Errorf("发送到WebSocket失败: %v", err)
					cancel() // 通知其他goroutine退出
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					logx.Errorf("读取PTY失败: %v", err)
				}
				cancel() // 通知其他goroutine退出
				return
			}
		}
	}()

	// WS -> 输入/resize
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 设置读取超时
			ws.SetReadDeadline(time.Now().Add(60 * time.Second))
			var msg WebSSHMessage
			if err := ws.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					logx.Errorf("读取WebSocket失败: %v", err)
				}
				cancel() // 通知其他goroutine退出
				return
			}
			switch msg.Type {
			case "input":
				if _, err := f.Write([]byte(msg.Data)); err != nil {
					logx.Errorf("写入PTY失败: %v", err)
					cancel()
					return
				}
			case "resize":
				if msg.Rows > 0 && msg.Cols > 0 {
					_ = pty.Setsize(f, &pty.Winsize{Rows: uint16(msg.Rows), Cols: uint16(msg.Cols)})
				}
			case "ping":
				// 心跳请求，返回pong
				ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(map[string]interface{}{"type": "pong"}); err != nil {
					logx.Errorf("发送pong失败: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// 等待退出
	_ = cmd.Wait()
	wg.Wait()
	return nil
}
