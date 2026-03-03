package monitor

import (
	"fmt"
	"regexp"
	"server/internal/server/service/deploy"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strconv"
	"strings"
	"time"
)

// BandwidthTestResult 带宽测速结果
type BandwidthTestResult struct {
	ServerId   int    `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerHost string `json:"server_host"`
	// 内网吞吐量测试（下载方向: 中继→App）
	InternalSpeeds []InternalSpeed `json:"internal_speeds"`
	// GOST 隧道上传吞吐量（上传方向: 中继→GOST隧道→App）
	GostUploadSpeeds []InternalSpeed `json:"gost_upload_speeds"`
	// 延迟测试
	Latencies []LatencyResult `json:"latencies"`
	// 公网带宽
	PublicUploadKBps   float64 `json:"public_upload_kbps"`
	PublicDownloadKBps float64 `json:"public_download_kbps"`
}

type InternalSpeed struct {
	Target   string  `json:"target"`
	SpeedMBs float64 `json:"speed_mbs"`
}

type LatencyResult struct {
	Target string  `json:"target"`
	AvgMs  float64 `json:"avg_ms"`
}

// RunBandwidthTest 通过 SSH 在中继服务器上跑测速
func RunBandwidthTest(serverId int, progressCallback func(string)) (*BandwidthTestResult, error) {
	// 1. 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", serverId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在: %d", serverId)
	}

	result := &BandwidthTestResult{
		ServerId:   serverId,
		ServerName: server.Name,
		ServerHost: server.Host,
	}

	progressCallback(fmt.Sprintf("开始测速: %s (%s)", server.Name, server.Host))

	// 2. SSH 连接
	progressCallback("步骤 1/4: 建立 SSH 连接...")
	sshClient, err := deploy.GetSSHClient(serverId)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer sshClient.Close()
	progressCallback("SSH 连接成功")

	// 3. 查找该中继关联的 App 节点
	progressCallback("步骤 2/4: 查找关联的 App 节点...")
	var gostServers []entity.MerchantGostServers
	_ = dbs.DBAdmin.Where("server_id = ? AND status = 1", serverId).Find(&gostServers)

	appIPs := make(map[string]bool)
	if len(gostServers) > 0 {
		merchantIds := make([]int, 0)
		for _, gs := range gostServers {
			merchantIds = append(merchantIds, gs.MerchantId)
		}
		var merchants []entity.Merchants
		_ = dbs.DBAdmin.In("id", merchantIds).Find(&merchants)
		for _, m := range merchants {
			if m.ServerIP != "" {
				appIPs[m.ServerIP] = true
			}
		}
	}

	if len(appIPs) == 0 {
		progressCallback("未找到关联的 App 节点，跳过内网测试")
	} else {
		progressCallback(fmt.Sprintf("找到 %d 个 App 节点", len(appIPs)))
	}

	// 4. 延迟测试 (TCP connect time，跨云环境 ICMP 可能被拦截)
	progressCallback("步骤 3/4: 延迟测试...")
	for ip := range appIPs {
		// 先尝试 ICMP ping
		progressCallback(fmt.Sprintf("  ping %s ...", ip))
		output, err := sshClient.ExecuteCommandWithTimeout(
			fmt.Sprintf("ping -c 3 -W 3 -q %s 2>&1", ip),
			15*time.Second,
		)
		if err == nil {
			avgMs := parsePingAvg(output)
			if avgMs > 0 {
				result.Latencies = append(result.Latencies, LatencyResult{
					Target: ip,
					AvgMs:  avgMs,
				})
				progressCallback(fmt.Sprintf("  ping %s: 平均延迟 %.1f ms", ip, avgMs))
				continue
			}
		}
		// ICMP 被拦截，改用 TCP connect time (多次取平均)
		progressCallback(fmt.Sprintf("  ICMP 不通，改用 TCP 测延迟 %s:82 ...", ip))
		output, err = sshClient.ExecuteCommandWithTimeout(
			fmt.Sprintf("for i in 1 2 3; do curl -o /dev/null -s -w '%%{time_connect}\\n' --connect-timeout 5 http://%s:82/ 2>/dev/null || true; done", ip),
			25*time.Second,
		)
		if err == nil {
			var totalMs float64
			var count int
			for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
				t := parseFloat(line)
				if t > 0 {
					totalMs += t * 1000
					count++
				}
			}
			if count > 0 {
				avgMs := totalMs / float64(count)
				result.Latencies = append(result.Latencies, LatencyResult{
					Target: ip,
					AvgMs:  avgMs,
				})
				progressCallback(fmt.Sprintf("  TCP 延迟 %s: 平均 %.1f ms", ip, avgMs))
			} else {
				progressCallback(fmt.Sprintf("  TCP 延迟测试 %s 失败", ip))
			}
		}
	}

	// 5. 跨云吞吐量测试 — 下载 App Web 大文件测速
	progressCallback("步骤 3.5/4: 中继→App 吞吐量测试...")
	for ip := range appIPs {
		progressCallback(fmt.Sprintf("  测试 中继 → %s 吞吐量...", ip))

		// 从 App Web 首页提取最大的 JS/CSS 资源 URL，然后下载测速
		progressCallback(fmt.Sprintf("    获取 %s:82 静态资源列表...", ip))
		indexOutput, err := sshClient.ExecuteCommandWithTimeout(
			fmt.Sprintf("curl -s --max-time 5 http://%s:82/ 2>/dev/null | grep -oE '(src|href)=\"[^\"]+\\.(js|css)\"' | sed 's/.*\"\\(.*\\)\"/\\1/' | sed 's/^\\.\\/*/\\//'", ip),
			10*time.Second,
		)
		if err != nil || strings.TrimSpace(indexOutput) == "" {
			progressCallback(fmt.Sprintf("    无法获取 %s:82 资源列表", ip))
			result.InternalSpeeds = append(result.InternalSpeeds, InternalSpeed{Target: ip, SpeedMBs: 0})
			continue
		}

		// 找到资源 URL，逐个尝试下载（先试 .js 大文件）
		assets := strings.Split(strings.TrimSpace(indexOutput), "\n")
		progressCallback(fmt.Sprintf("    找到 %d 个资源，开始下载测速...", len(assets)))

		var bestSpeed float64
		var bestAsset string
		for _, asset := range assets {
			asset = strings.TrimSpace(asset)
			if asset == "" {
				continue
			}
			url := fmt.Sprintf("http://%s:82%s", ip, asset)
			output, err := sshClient.ExecuteCommandWithTimeout(
				fmt.Sprintf("curl -o /dev/null -s -w '%%{speed_download} %%{size_download}' --max-time 15 '%s' 2>/dev/null || echo '0 0'", url),
				20*time.Second,
			)
			if err != nil {
				continue
			}
			parts := strings.Fields(strings.TrimSpace(output))
			if len(parts) < 2 {
				continue
			}
			speed := parseFloat(parts[0])  // bytes/sec
			size := parseFloat(parts[1])   // bytes
			if size > 100000 && speed > bestSpeed { // 文件 > 100KB 才有参考价值
				bestSpeed = speed
				bestAsset = asset
				speedMB := speed / 1024 / 1024
				progressCallback(fmt.Sprintf("    下载 %s (%.0f KB): %.2f MB/s", asset, size/1024, speedMB))
			}
		}

		if bestSpeed > 0 {
			speedMB := bestSpeed / 1024 / 1024
			result.InternalSpeeds = append(result.InternalSpeeds, InternalSpeed{
				Target:   fmt.Sprintf("%s (%s)", ip, bestAsset),
				SpeedMBs: speedMB,
			})
			if speedMB >= 1 {
				progressCallback(fmt.Sprintf("    → 最佳吞吐量: %.2f MB/s (%.1f Mbps)", speedMB, speedMB*8))
			} else {
				progressCallback(fmt.Sprintf("    → 最佳吞吐量: %.0f KB/s (%.1f Mbps)", bestSpeed/1024, bestSpeed*8/1024/1024))
			}
		} else {
			progressCallback(fmt.Sprintf("  → %s: 无足够大的资源可测速", ip))
			result.InternalSpeeds = append(result.InternalSpeeds, InternalSpeed{Target: ip, SpeedMBs: 0})
		}
	}

	// 5.5 中继→App 上传吞吐量测试（直接上传到 App 节点，测实际网络上传速度）
	if len(appIPs) > 0 {
		progressCallback("步骤 3.75/4: 中继→App 上传吞吐量测试...")

		// 在中继上生成 1MB 测试文件
		_, _ = sshClient.ExecuteCommandWithTimeout(
			"dd if=/dev/zero of=/tmp/gost_ul_test bs=1M count=1 2>/dev/null",
			10*time.Second,
		)

		for ip := range appIPs {
			progressCallback(fmt.Sprintf("  测试 中继 → %s 上传吞吐量...", ip))

			output, err := sshClient.ExecuteCommandWithTimeout(
				fmt.Sprintf("curl -X POST --data-binary @/tmp/gost_ul_test -o /dev/null -s -w '%%{speed_upload}' --max-time 30 http://%s:82/ 2>/dev/null || echo 0", ip),
				35*time.Second,
			)
			if err != nil {
				progressCallback(fmt.Sprintf("    上传测试失败: %v", err))
				result.GostUploadSpeeds = append(result.GostUploadSpeeds, InternalSpeed{
					Target: ip, SpeedMBs: 0,
				})
				continue
			}

			speed := parseFloat(strings.TrimSpace(output)) // bytes/sec
			speedMB := speed / 1024 / 1024
			result.GostUploadSpeeds = append(result.GostUploadSpeeds, InternalSpeed{
				Target:   ip,
				SpeedMBs: speedMB,
			})
			if speed > 0 {
				if speedMB >= 1 {
					progressCallback(fmt.Sprintf("    → 上传吞吐量: %.2f MB/s (%.1f Mbps)", speedMB, speedMB*8))
				} else {
					progressCallback(fmt.Sprintf("    → 上传吞吐量: %.0f KB/s (%.1f Mbps)", speed/1024, speed*8/1024/1024))
				}
			} else {
				progressCallback("    → 上传吞吐量: 未能获取有效速度")
			}
		}

		// 不清理，下一步 GOST MinIO 测试还要用
	}

	// 5.75 GOST 隧道→MinIO 上传吞吐量测试（通过 GOST mTLS 隧道到 MinIO 端口）
	if len(gostServers) > 0 {
		progressCallback("步骤 3.8/4: GOST 隧道→MinIO 上传测试...")

		// 构建 merchantId → merchant 映射
		merchantMap := make(map[int]*entity.Merchants)
		{
			merchantIds := make([]int, 0)
			for _, gs := range gostServers {
				merchantIds = append(merchantIds, gs.MerchantId)
			}
			var merchants []entity.Merchants
			_ = dbs.DBAdmin.In("id", merchantIds).Find(&merchants)
			for i := range merchants {
				merchantMap[merchants[i].Id] = &merchants[i]
			}
		}

		// 确保测试文件存在
		_, _ = sshClient.ExecuteCommandWithTimeout(
			"test -f /tmp/gost_ul_test || dd if=/dev/zero of=/tmp/gost_ul_test bs=1M count=1 2>/dev/null",
			10*time.Second,
		)

		for _, gs := range gostServers {
			basePort := gs.ListenPort
			if basePort == 0 {
				if m, ok := merchantMap[gs.MerchantId]; ok {
					basePort = m.Port
				}
			}
			if basePort == 0 {
				continue
			}

			merchantName := fmt.Sprintf("商户%d", gs.MerchantId)
			if m, ok := merchantMap[gs.MerchantId]; ok {
				merchantName = m.Name
			}

			minioPort := basePort + 3 // PortOffsetMinIO
			progressCallback(fmt.Sprintf("  测试 GOST 隧道 → %s MinIO (端口 %d)...", merchantName, minioPort))

			// -H 'Expect:' 禁用 100-continue，确保 curl 直接发送数据
			// 用 speed_upload 和 time_total 双重判断：time_total < 0.5s 说明没走隧道（命中本地服务）
			output, err := sshClient.ExecuteCommandWithTimeout(
				fmt.Sprintf("curl -X PUT -T /tmp/gost_ul_test -H 'Expect:' -o /dev/null -s -w '%%{speed_upload} %%{time_total}' --max-time 30 http://127.0.0.1:%d/test/speed-test 2>/dev/null || echo '0 0'", minioPort),
				35*time.Second,
			)
			if err != nil {
				progressCallback(fmt.Sprintf("    GOST MinIO 上传测试失败: %v", err))
				result.GostUploadSpeeds = append(result.GostUploadSpeeds, InternalSpeed{
					Target: fmt.Sprintf("GOST→MinIO (%s)", merchantName), SpeedMBs: 0,
				})
				continue
			}

			parts := strings.Fields(strings.TrimSpace(output))
			speed := float64(0)
			timeTotal := float64(0)
			if len(parts) >= 1 {
				speed = parseFloat(parts[0]) // bytes/sec
			}
			if len(parts) >= 2 {
				timeTotal = parseFloat(parts[1]) // seconds
			}

			// 1MB 上传 < 0.5s 完成，说明端口命中本地服务而非 GOST 隧道
			if timeTotal > 0 && timeTotal < 0.5 {
				progressCallback(fmt.Sprintf("    → 端口 %d 命中本地服务 (耗时 %.3fs)，非 GOST 隧道，跳过", minioPort, timeTotal))
				continue
			}

			speedMB := speed / 1024 / 1024
			result.GostUploadSpeeds = append(result.GostUploadSpeeds, InternalSpeed{
				Target:   fmt.Sprintf("GOST→MinIO (%s)", merchantName),
				SpeedMBs: speedMB,
			})
			if speed > 0 {
				if speedMB >= 1 {
					progressCallback(fmt.Sprintf("    → GOST MinIO 上传: %.2f MB/s (%.1f Mbps)", speedMB, speedMB*8))
				} else {
					progressCallback(fmt.Sprintf("    → GOST MinIO 上传: %.0f KB/s (%.1f Mbps)", speed/1024, speed*8/1024/1024))
				}
			} else {
				progressCallback("    → GOST MinIO 上传: 未能获取有效速度 (端口可能未开放)")
			}
		}
	}

	// 清理临时文件
	_, _ = sshClient.ExecuteCommandWithTimeout("rm -f /tmp/gost_ul_test", 5*time.Second)

	// 6. 公网带宽测试
	progressCallback("步骤 4/4: 公网带宽测试...")

	// 下载速度 (使用 speedtest 小文件)
	progressCallback("  测试下载速度...")
	dlOutput, err := sshClient.ExecuteCommandWithTimeout(
		"curl -o /dev/null -w '%{speed_download}' --max-time 15 https://speed.cloudflare.com/__down?bytes=5000000 2>/dev/null || echo 0",
		20*time.Second,
	)
	if err == nil {
		dlSpeed := parseFloat(strings.TrimSpace(dlOutput))
		result.PublicDownloadKBps = dlSpeed / 1024
		progressCallback(fmt.Sprintf("  下载速度: %.0f KB/s (%.1f Mbps)", dlSpeed/1024, dlSpeed*8/1024/1024))
	} else {
		progressCallback(fmt.Sprintf("  下载测试失败: %v", err))
	}

	// 上传速度 (先生成临时文件，再用 curl 上传)
	progressCallback("  测试上传速度...")
	ulOutput, err := sshClient.ExecuteCommandWithTimeout(
		"dd if=/dev/zero of=/tmp/ul_test bs=1M count=3 2>/dev/null && curl -X POST -H 'Content-Type: application/octet-stream' --data-binary @/tmp/ul_test -o /dev/null -s -w '%{speed_upload}' --max-time 20 https://speed.cloudflare.com/__up 2>/dev/null; rm -f /tmp/ul_test",
		30*time.Second,
	)
	if err == nil {
		ulSpeed := parseFloat(strings.TrimSpace(ulOutput))
		if ulSpeed > 0 {
			result.PublicUploadKBps = ulSpeed / 1024
			progressCallback(fmt.Sprintf("  上传速度: %.0f KB/s (%.1f Mbps)", ulSpeed/1024, ulSpeed*8/1024/1024))
		} else {
			progressCallback("  上传测试: 未能获取有效速度")
		}
	} else {
		progressCallback(fmt.Sprintf("  上传测试失败: %v", err))
	}

	// 总结
	progressCallback("========== 测速完成 ==========")
	if result.PublicDownloadKBps > 0 {
		progressCallback(fmt.Sprintf("公网下载: %.0f KB/s (%.1f Mbps)", result.PublicDownloadKBps, result.PublicDownloadKBps*8/1024))
	}
	if result.PublicUploadKBps > 0 {
		progressCallback(fmt.Sprintf("公网上传: %.0f KB/s (%.1f Mbps)", result.PublicUploadKBps, result.PublicUploadKBps*8/1024))
	}
	for _, l := range result.Latencies {
		progressCallback(fmt.Sprintf("延迟 → %s: %.1f ms", l.Target, l.AvgMs))
	}
	for _, s := range result.InternalSpeeds {
		progressCallback(fmt.Sprintf("下载吞吐量 → %s: %.2f MB/s", s.Target, s.SpeedMBs))
	}
	for _, s := range result.GostUploadSpeeds {
		if s.SpeedMBs >= 1 {
			progressCallback(fmt.Sprintf("上传吞吐量 → %s: %.2f MB/s", s.Target, s.SpeedMBs))
		} else if s.SpeedMBs > 0 {
			progressCallback(fmt.Sprintf("上传吞吐量 → %s: %.0f KB/s", s.Target, s.SpeedMBs*1024))
		}
	}

	return result, nil
}

// parsePingAvg 从 ping -q 输出提取平均延迟
func parsePingAvg(output string) float64 {
	// rtt min/avg/max/mdev = 1.234/2.345/3.456/0.567 ms
	re := regexp.MustCompile(`=\s*[\d.]+/([\d.]+)/`)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		return parseFloat(match[1])
	}
	return 0
}

// parseDdSpeed 从 dd 输出提取传输速度 (MB/s)
func parseDdSpeed(output string) float64 {
	// dd output: "5242880 bytes (5.2 MB, 5.0 MiB) copied, 0.256897 s, 20.4 MB/s"
	re := regexp.MustCompile(`([\d.]+)\s*[MGK]B/s`)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		speed := parseFloat(match[1])
		if strings.Contains(output, "GB/s") {
			speed *= 1024
		} else if strings.Contains(output, "KB/s") {
			speed /= 1024
		}
		return speed
	}
	return 0
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	// Remove single quotes that curl may wrap around
	s = strings.Trim(s, "'")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
