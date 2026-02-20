package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	Host          string
	Port          int
	Username      string
	Password      string
	PrivateKey    string
	Client        *ssh.Client   // 导出字段，供外部访问
	stopKeepAlive chan struct{} // 用于停止 keepAlive goroutine
	keepAliveOnce sync.Once     // 确保只启动一次 keepAlive
}

// isLocal 判断是否为本地地址
func (c *SSHClient) isLocal() bool {
	host := strings.TrimSpace(strings.ToLower(c.Host))
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// runLocalCommand 执行本地命令，支持可选超时
func runLocalCommand(cmd string, timeout time.Duration) (string, error) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		// 使用 PowerShell 非交互模式，避免加载 profile 带来的副作用
		c = exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-NonInteractive", "-Command", cmd)
	} else {
		// 使用 bash -c（非登录/非交互），并强制 C locale，避免 setlocale 警告
		c = exec.CommandContext(ctx, "bash", "-c", cmd)
		c.Env = append(os.Environ(),
			"LC_ALL=C",
			"LANG=C",
			"LANGUAGE=C",
			"TERM=xterm-256color",
		)
	}
	out, err := c.CombinedOutput()
	return string(out), err
}

// Connect 连接到SSH服务器
func (c *SSHClient) Connect() error {
	if c.isLocal() {
		// 本地模式无需建立SSH连接
		return nil
	}
	var authMethod ssh.AuthMethod

	if c.PrivateKey != "" {
		// 使用私钥认证
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
		if err != nil {
			return fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		// 使用密码认证
		authMethod = ssh.Password(c.Password)
	}

	config := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证host key
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %v", err)
	}

	c.Client = client
	c.stopKeepAlive = make(chan struct{})

	// 启动后台 goroutine 定期发送保活包（确保只启动一次）
	c.keepAliveOnce.Do(func() {
		go c.keepAlive()
	})

	return nil
}

// keepAlive 定期发送保活包，保持SSH连接活跃
func (c *SSHClient) keepAlive() {
	if c.Client == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second) // 每30秒发送一次保活包
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.Client == nil {
				return
			}

			// 发送 keepalive 请求
			_, _, err := c.Client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				// 连接可能已断开，停止保活
				return
			}

		case <-c.stopKeepAlive:
			// 收到停止信号，退出 goroutine
			return
		}
	}
}

// ExecuteCommand 执行命令
func (c *SSHClient) ExecuteCommand(cmd string) (string, error) {
	// 本地执行分支：不经过SSH
	if c.isLocal() {
		return runLocalCommand(cmd, 0)
	}

	if c.Client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	session, err := c.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(cmd)

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	return output, err
}

// ExecuteCommandWithTimeout 带超时的命令执行
func (c *SSHClient) ExecuteCommandWithTimeout(cmd string, timeout time.Duration) (string, error) {
	// 本地执行支持超时，直接使用本地上下文
	if c.isLocal() {
		return runLocalCommand(cmd, timeout)
	}
	type result struct {
		output string
		err    error
	}

	ch := make(chan result, 1)
	go func() {
		output, err := c.ExecuteCommand(cmd)
		ch <- result{output, err}
	}()

	select {
	case res := <-ch:
		return res.output, res.err
	case <-time.After(timeout):
		return "", fmt.Errorf("命令执行超时(%v)", timeout)
	}
}

// ExecuteCommandSilent 执行命令（忽略错误）
func (c *SSHClient) ExecuteCommandSilent(cmd string) string {
	output, _ := c.ExecuteCommand(cmd)
	return output
}

// UploadFile 上传文件到远程服务器
func (c *SSHClient) UploadFile(remotePath string, content io.Reader) error {
	// 本地执行分支：直接写入文件
	if c.isLocal() {
		dir := filepath.Dir(remotePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		f, err := os.Create(remotePath)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(f, content); err != nil {
			return err
		}
		return nil
	}

	if c.Client == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	session, err := c.Client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, content)
	}()

	cmd := fmt.Sprintf("cat > %s", remotePath)
	return session.Run(cmd)
}

// ExecuteCommandStream 执行命令并返回 stdout 流式读取器（用于大输出场景，避免全部加载到内存）
// 调用方必须在读取完毕后调用 session.Wait() 和 session.Close()
func (c *SSHClient) ExecuteCommandStream(cmd string) (io.Reader, *ssh.Session, error) {
	if c.Client == nil {
		if err := c.Connect(); err != nil {
			return nil, nil, err
		}
	}

	session, err := c.Client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("创建会话失败: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, nil, fmt.Errorf("获取stdout管道失败: %v", err)
	}

	if err := session.Start(cmd); err != nil {
		session.Close()
		return nil, nil, fmt.Errorf("启动命令失败: %v", err)
	}

	return stdout, session, nil
}

// Close 关闭连接
func (c *SSHClient) Close() error {
	// 先停止 keepAlive goroutine
	if c.stopKeepAlive != nil {
		close(c.stopKeepAlive)
		c.stopKeepAlive = nil
	}

	// 再关闭 SSH 连接
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}

// IsConnected 检查是否已连接
func (c *SSHClient) IsConnected() bool {
	if c.isLocal() {
		return true
	}
	return c.Client != nil
}

// TestSSHConnection 测试SSH连接
func TestSSHConnection(host string, port int, username, password, privateKey string) error {
	client := &SSHClient{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		PrivateKey: privateKey,
	}

	// 本地直接返回成功（或执行一个本地命令验证）
	if client.isLocal() {
		_, _ = runLocalCommand("echo 'local connection test success'", 3*time.Second)
		return nil
	}

	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	// 执行简单命令测试
	_, err := client.ExecuteCommand("echo 'SSH connection test success'")
	return err
}

// ========== SSH 连接池 ==========

// PooledSSHClient 连接池中的SSH客户端
type PooledSSHClient struct {
	*SSHClient
	lastUsed time.Time
	mu       sync.Mutex
}

// SSHConnectionPool SSH连接池
type SSHConnectionPool struct {
	connections sync.Map // key: serverId(string), value: *PooledSSHClient
	idleTimeout time.Duration
}

var (
	// 全局连接池实例
	sshPool     *SSHConnectionPool
	poolOnce    sync.Once
	poolCleaner *time.Ticker
)

// GetSSHPool 获取全局SSH连接池
func GetSSHPool() *SSHConnectionPool {
	poolOnce.Do(func() {
		sshPool = &SSHConnectionPool{
			idleTimeout: 10 * time.Minute, // 10分钟空闲超时
		}
		// 启动清理goroutine
		poolCleaner = time.NewTicker(5 * time.Minute)
		go sshPool.cleanIdleConnections()
	})
	return sshPool
}

// GetOrCreateConnection 获取或创建SSH连接
func (p *SSHConnectionPool) GetOrCreateConnection(key string, host string, port int, username, password, privateKey string) (*PooledSSHClient, error) {
	// 尝试从池中获取现有连接
	if val, ok := p.connections.Load(key); ok {
		pooledClient := val.(*PooledSSHClient)
		pooledClient.mu.Lock()
		defer pooledClient.mu.Unlock()

		// 检查连接是否有效
		if pooledClient.IsConnected() && pooledClient.checkConnection() {
			pooledClient.lastUsed = time.Now()
			return pooledClient, nil
		}

		// 连接失效，尝试重连
		if pooledClient.isLocal() {
			// 本地模式无需重连
			pooledClient.lastUsed = time.Now()
			return pooledClient, nil
		}

		// 先关闭旧连接（停止 keepAlive goroutine）
		pooledClient.Close()

		// 重置 sync.Once 以允许重新启动 keepAlive
		pooledClient.keepAliveOnce = sync.Once{}

		if err := pooledClient.Connect(); err == nil {
			pooledClient.lastUsed = time.Now()
			return pooledClient, nil
		}

		// 重连失败，移除旧连接
		p.connections.Delete(key)
	}

	// 创建新连接
	client := &SSHClient{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		PrivateKey: privateKey,
	}

	// 本地模式跳过 SSH 连接
	if !client.isLocal() {
		if err := client.Connect(); err != nil {
			return nil, err
		}
	}

	pooledClient := &PooledSSHClient{
		SSHClient: client,
		lastUsed:  time.Now(),
	}

	p.connections.Store(key, pooledClient)
	return pooledClient, nil
}

// checkConnection 检查连接是否有效
func (pc *PooledSSHClient) checkConnection() bool {
	if pc.SSHClient != nil && pc.SSHClient.isLocal() {
		return true
	}
	if pc.Client == nil {
		return false
	}

	// 尝试执行一个简单命令来测试连接
	session, err := pc.Client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	// 执行简单命令
	err = session.Run("echo test")
	return err == nil
}

// RemoveConnection 移除连接
func (p *SSHConnectionPool) RemoveConnection(key string) {
	if val, ok := p.connections.LoadAndDelete(key); ok {
		pooledClient := val.(*PooledSSHClient)
		pooledClient.Close()
	}
}

// cleanIdleConnections 清理空闲连接
func (p *SSHConnectionPool) cleanIdleConnections() {
	for range poolCleaner.C {
		now := time.Now()
		p.connections.Range(func(key, value interface{}) bool {
			pooledClient := value.(*PooledSSHClient)
			pooledClient.mu.Lock()
			defer pooledClient.mu.Unlock()

			// 如果连接空闲时间超过设定值，关闭并移除
			if now.Sub(pooledClient.lastUsed) > p.idleTimeout {
				pooledClient.Close()
				p.connections.Delete(key)
			}
			return true
		})
	}
}

// CloseAll 关闭所有连接
func (p *SSHConnectionPool) CloseAll() {
	p.connections.Range(func(key, value interface{}) bool {
		pooledClient := value.(*PooledSSHClient)
		pooledClient.Close()
		p.connections.Delete(key)
		return true
	})
}
