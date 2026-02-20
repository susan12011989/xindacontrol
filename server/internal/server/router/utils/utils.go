package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"server/internal/server/middleware"
	"server/internal/server/model"
	"server/pkg/result"
	"strings"
	"time"

	"server/internal/server/service/utils"

	"github.com/gin-gonic/gin"
)

func Routes(gi gin.IRouter) {
	group := gi.Group("utils", middleware.Authorization)
	group.POST("port2enterprise", port2enterprise) // 端口转企业号
	group.POST("enterprise2port", enterprise2port) // 企业号转端口
	group.GET("check-port", checkPortAvailable)    // 校验端口是否空闲
	group.POST("embedips", embedIPs)               // 将IP列表嵌入文件
	group.POST("extractips", extractIPs)           // 从文件提取IP列表
	group.POST("embedurls", embedURLs)             // 将URL列表嵌入文件
	group.POST("extracturls", extractURLs)         // 从文件提取URL列表
	group.POST("embedips-batch", embedIPsBatch)    // 批量将IP列表嵌入文件
	group.POST("embedurls-batch", embedURLsBatch)  // 批量将URL列表嵌入文件
	group.POST("generateversion", generateVersion) // 生成版本配置文件
	group.POST("decryptversion", decryptVersion)   // 解密版本配置文件
}

func port2enterprise(c *gin.Context) {
	var req model.Port2EnterpriseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	result.GOK(c, model.Port2EnterpriseResp{
		Enterprise: utils.Port2Enterprise(req.Port),
	})
}

func enterprise2port(c *gin.Context) {
	var req model.Enterprise2PortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	port, err := utils.Enterprise2Port(req.Enterprise)
	if err != nil {
		result.GParamErr(c, err)
		return
	}
	result.GOK(c, model.Enterprise2PortResp{
		Port: port,
	})
}

// checkPortAvailable 校验端口及端口+1是否被占用
func checkPortAvailable(c *gin.Context) {
	var req struct {
		Port int `form:"port" json:"port" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	// 端口唯一性检查已移除：每个商户配独立系统服务器+隧道，端口可复用
	result.GOK(c, gin.H{"ok": true})
}

// embedIPs 将IP列表嵌入到文件中
func embedIPs(c *gin.Context) {
	// 1. 解析表单数据
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 获取IP列表（从 FormData 中获取多个 ips 字段）
	ips := c.Request.Form["ips"]
	if len(ips) == 0 {
		result.GParamErr(c, fmt.Errorf("IP列表不能为空"))
		return
	}

	// 验证IP格式
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			result.GParamErr(c, fmt.Errorf("无效的IP地址: %s", ip))
			return
		}
	}

	// 固定seed
	const seed uint32 = 444013

	// 3. 保存上传的文件到临时目录
	tmpDir := os.TempDir()
	srcPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), header.Filename))
	dstPath := filepath.Join(tmpDir, fmt.Sprintf("output_%d_%s", time.Now().UnixNano(), header.Filename))

	// 保存上传文件
	out, err := os.Create(srcPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(srcPath)
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 4. 调用嵌入函数
	err = utils.EmbedIntoFile(srcPath, dstPath, ips, seed)
	if err != nil {
		os.Remove(srcPath)
		result.GErr(c, fmt.Errorf("嵌入IP失败: %v", err))
		return
	}

	// 5. 读取处理后的文件
	defer func() {
		os.Remove(srcPath)
		os.Remove(dstPath)
	}()

	// 6. 返回文件
	c.FileAttachment(dstPath, header.Filename)
}

// extractIPs 从文件中提取IP列表
func extractIPs(c *gin.Context) {
	// 1. 解析表单数据
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 固定seed
	const seed uint32 = 444013

	// 3. 保存上传的文件到临时目录
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("extract_%d", time.Now().UnixNano()))

	out, err := os.Create(tmpPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 4. 提取IP列表
	defer os.Remove(tmpPath)

	ips, err := utils.ExtractFromFile(tmpPath, seed)
	if err != nil {
		result.GErr(c, fmt.Errorf("提取IP失败: %v", err))
		return
	}

	// 5. 返回IP列表
	result.GOK(c, model.ExtractIPsResp{
		IPs: ips,
	})
}

// embedURLs 将URL列表嵌入到文件中
func embedURLs(c *gin.Context) {
	// 1. 解析表单数据
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 获取URL列表
	urls := c.Request.Form["urls"]
	if len(urls) == 0 {
		result.GParamErr(c, fmt.Errorf("URL列表不能为空"))
		return
	}
	// 基础校验
	for _, u := range urls {
		if len(u) == 0 {
			result.GParamErr(c, fmt.Errorf("存在空URL"))
			return
		}
		if parsed, err := url.ParseRequestURI(u); err != nil || parsed.Scheme == "" || parsed.Host == "" {
			result.GParamErr(c, fmt.Errorf("无效的URL: %s", u))
			return
		}
	}

	// 固定seed
	const seed uint32 = 444013

	// 3. 保存上传的文件到临时目录
	tmpDir := os.TempDir()
	srcPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), header.Filename))
	dstPath := filepath.Join(tmpDir, fmt.Sprintf("output_%d_%s", time.Now().UnixNano(), header.Filename))

	out, err := os.Create(srcPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(srcPath)
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 4. 调用嵌入函数
	err = utils.EmbedURLsIntoFile(srcPath, dstPath, urls, seed)
	if err != nil {
		os.Remove(srcPath)
		result.GErr(c, fmt.Errorf("嵌入URL失败: %v", err))
		return
	}

	// 5. 清理与返回
	defer func() {
		os.Remove(srcPath)
		os.Remove(dstPath)
	}()
	c.FileAttachment(dstPath, header.Filename)
}

// extractURLs 从文件中提取URL列表
func extractURLs(c *gin.Context) {
	// 1. 解析表单数据
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 固定seed
	const seed uint32 = 444013

	// 3. 保存上传的文件到临时目录
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("extract_%d", time.Now().UnixNano()))

	out, err := os.Create(tmpPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 4. 提取URL列表
	defer os.Remove(tmpPath)
	urls, err := utils.ExtractURLsFromFile(tmpPath, seed)
	if err != nil {
		result.GErr(c, fmt.Errorf("提取URL失败: %v", err))
		return
	}

	// 5. 返回URL列表
	result.GOK(c, model.ExtractURLsResp{URLs: urls})
}

// generateVersion 生成并加密版本配置文件
func generateVersion(c *gin.Context) {
	// 1. 解析请求
	var req model.GenerateVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}

	// 2. 固定密钥
	const secretHex = "3d4270b340f381cfd70b8bed30c3191845e52a45c9aec15e3e83de4500761af4"
	key, err := utils.ParseHexKey(secretHex)
	if err != nil {
		result.GErr(c, fmt.Errorf("解析密钥失败: %v", err))
		return
	}

	// 3. 生成当前时间
	now := time.Now()
	updatedAt := now.Format("2006-01-02 15:04:05")

	// 4. 转换版本列表
	versions := make([]utils.VersionEntry, len(req.Versions))
	for i, v := range req.Versions {
		versions[i] = utils.VersionEntry{
			Channel: v.Channel,
			Version: v.Version,
		}
	}

	// 5. 生成临时文件路径
	tmpDir := os.TempDir()
	jsonPath := filepath.Join(tmpDir, fmt.Sprintf("content_%d.json", now.UnixNano()))
	txtPath := filepath.Join(tmpDir, fmt.Sprintf("content_%d.txt", now.UnixNano()))
	defer func() {
		os.Remove(jsonPath)
		os.Remove(txtPath)
	}()

	// 6. 查找HTML文件（从工作目录）
	var (
		termsPath      = "terms.html"
		privacyPath    = "privacy.html"
		termsPrePath   = "terms_pre.html"
		privacyPrePath = "privacy_pre.html"
	)
	switch req.Package {
	case "mida":
		termsPath = "./mida/terms.html"
		privacyPath = "./mida/privacy.html"
		termsPrePath = "./mida/terms_pre.html"
		privacyPrePath = "./mida/privacy_pre.html"
	case "mihangyan":
		termsPath = "./mihangyan/terms.html"
		privacyPath = "./mihangyan/privacy.html"
		termsPrePath = "./mihangyan/terms_pre.html"
		privacyPrePath = "./mihangyan/privacy_pre.html"
	}

	// 7. 生成content.json（使用service层函数）
	err = utils.GenerateContentJSONFromData(updatedAt, versions, termsPath, privacyPath, termsPrePath, privacyPrePath, jsonPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("生成JSON失败: %v", err))
		return
	}

	// 8. 加密生成content.txt
	err = utils.EncryptFile(jsonPath, txtPath, key)
	if err != nil {
		result.GErr(c, fmt.Errorf("加密文件失败: %v", err))
		return
	}

	// 9. 返回加密后的文件
	c.FileAttachment(txtPath, "content.txt")
}

// decryptVersion 解密版本配置文件
func decryptVersion(c *gin.Context) {
	// 1. 解析表单数据
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 固定密钥
	const secretHex = "3d4270b340f381cfd70b8bed30c3191845e52a45c9aec15e3e83de4500761af4"
	key, err := utils.ParseHexKey(secretHex)
	if err != nil {
		result.GErr(c, fmt.Errorf("解析密钥失败: %v", err))
		return
	}

	// 3. 保存上传的文件到临时目录
	tmpDir := os.TempDir()
	txtPath := filepath.Join(tmpDir, fmt.Sprintf("encrypted_%d.txt", time.Now().UnixNano()))
	jsonPath := filepath.Join(tmpDir, fmt.Sprintf("decrypted_%d.json", time.Now().UnixNano()))
	defer func() {
		os.Remove(txtPath)
		os.Remove(jsonPath)
	}()

	// 保存上传文件
	out, err := os.Create(txtPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 4. 解密文件
	err = utils.DecryptFile(txtPath, jsonPath, key)
	if err != nil {
		result.GErr(c, fmt.Errorf("解密失败: %v", err))
		return
	}

	// 5. 返回解密后的JSON文件
	c.FileAttachment(jsonPath, "content.json")
}

// embedIPsBatch 批量将IP列表嵌入到zip文件中的所有文件
func embedIPsBatch(c *gin.Context) {
	// 1. 解析表单数据 - 获取zip文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 验证文件是否为zip格式
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		result.GParamErr(c, fmt.Errorf("请上传zip格式的文件"))
		return
	}

	// 3. 获取IP列表
	ips := c.Request.Form["ips"]
	if len(ips) == 0 {
		result.GParamErr(c, fmt.Errorf("IP列表不能为空"))
		return
	}

	// 验证IP格式
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			result.GParamErr(c, fmt.Errorf("无效的IP地址: %s", ip))
			return
		}
	}

	// 固定seed
	const seed uint32 = 444013

	// 4. 创建临时目录
	tmpDir := os.TempDir()
	workDir := filepath.Join(tmpDir, fmt.Sprintf("embed_work_%d", time.Now().UnixNano()))
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d.zip", time.Now().UnixNano()))
	extractDir := filepath.Join(workDir, "extract")
	outputDir := filepath.Join(workDir, "output")
	outputZipPath := filepath.Join(tmpDir, fmt.Sprintf("result_%d.zip", time.Now().UnixNano()))

	defer func() {
		os.RemoveAll(workDir)
		os.Remove(zipPath)
		os.Remove(outputZipPath)
	}()

	// 创建工作目录
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		result.GErr(c, fmt.Errorf("创建临时目录失败: %v", err))
		return
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.GErr(c, fmt.Errorf("创建输出目录失败: %v", err))
		return
	}

	// 5. 保存上传的zip文件
	out, err := os.Create(zipPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 6. 解压缩zip文件
	if err := unzipFile(zipPath, extractDir); err != nil {
		result.GErr(c, fmt.Errorf("解压文件失败: %v", err))
		return
	}

	// 7. 遍历解压后的文件，逐个嵌入IP
	err = filepath.Walk(extractDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(extractDir, srcPath)
		if err != nil {
			return err
		}

		// 忽略 __MACOSX 目录及其内容
		if strings.HasPrefix(relPath, "__MACOSX") || strings.Contains(relPath, string(os.PathSeparator)+"__MACOSX") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过目录本身（但不跳过遍历）
		if info.IsDir() {
			return nil
		}

		// 目标文件路径
		dstPath := filepath.Join(outputDir, relPath)

		// 确保目标目录存在
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}

		// 执行嵌入操作
		if err := utils.EmbedIntoFile(srcPath, dstPath, ips, seed); err != nil {
			return fmt.Errorf("嵌入文件 %s 失败: %v", relPath, err)
		}

		return nil
	})

	if err != nil {
		result.GErr(c, fmt.Errorf("批量处理失败: %v", err))
		return
	}

	// 8. 压缩输出目录
	if err := zipDirectory(outputDir, outputZipPath); err != nil {
		result.GErr(c, fmt.Errorf("压缩结果失败: %v", err))
		return
	}

	// 9. 返回结果zip文件
	resultFilename := strings.TrimSuffix(header.Filename, ".zip") + "_embedded.zip"
	c.FileAttachment(outputZipPath, resultFilename)
}

// embedURLsBatch 批量将URL列表嵌入到zip文件中的所有文件
func embedURLsBatch(c *gin.Context) {
	// 1. 解析表单数据 - 获取zip文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, fmt.Errorf("文件上传失败: %v", err))
		return
	}
	defer file.Close()

	// 2. 验证文件是否为zip格式
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		result.GParamErr(c, fmt.Errorf("请上传zip格式的文件"))
		return
	}

	// 3. 获取URL列表
	urls := c.Request.Form["urls"]
	if len(urls) == 0 {
		result.GParamErr(c, fmt.Errorf("URL列表不能为空"))
		return
	}

	// 基础校验
	for _, u := range urls {
		if len(u) == 0 {
			result.GParamErr(c, fmt.Errorf("存在空URL"))
			return
		}
		if parsed, err := url.ParseRequestURI(u); err != nil || parsed.Scheme == "" || parsed.Host == "" {
			result.GParamErr(c, fmt.Errorf("无效的URL: %s", u))
			return
		}
	}

	// 固定seed
	const seed uint32 = 444013

	// 4. 创建临时目录
	tmpDir := os.TempDir()
	workDir := filepath.Join(tmpDir, fmt.Sprintf("embed_url_work_%d", time.Now().UnixNano()))
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d.zip", time.Now().UnixNano()))
	extractDir := filepath.Join(workDir, "extract")
	outputDir := filepath.Join(workDir, "output")
	outputZipPath := filepath.Join(tmpDir, fmt.Sprintf("result_%d.zip", time.Now().UnixNano()))

	defer func() {
		os.RemoveAll(workDir)
		os.Remove(zipPath)
		os.Remove(outputZipPath)
	}()

	// 创建工作目录
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		result.GErr(c, fmt.Errorf("创建临时目录失败: %v", err))
		return
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.GErr(c, fmt.Errorf("创建输出目录失败: %v", err))
		return
	}

	// 5. 保存上传的zip文件
	out, err := os.Create(zipPath)
	if err != nil {
		result.GErr(c, fmt.Errorf("创建临时文件失败: %v", err))
		return
	}
	_, err = io.Copy(out, file)
	out.Close()
	if err != nil {
		result.GErr(c, fmt.Errorf("保存文件失败: %v", err))
		return
	}

	// 6. 解压缩zip文件
	if err := unzipFile(zipPath, extractDir); err != nil {
		result.GErr(c, fmt.Errorf("解压文件失败: %v", err))
		return
	}

	// 7. 遍历解压后的文件，逐个嵌入URL
	err = filepath.Walk(extractDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(extractDir, srcPath)
		if err != nil {
			return err
		}

		// 忽略 __MACOSX 目录及其内容
		if strings.HasPrefix(relPath, "__MACOSX") || strings.Contains(relPath, string(os.PathSeparator)+"__MACOSX") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过目录本身（但不跳过遍历）
		if info.IsDir() {
			return nil
		}

		// 目标文件路径
		dstPath := filepath.Join(outputDir, relPath)

		// 确保目标目录存在
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %v", err)
		}

		// 执行嵌入操作
		if err := utils.EmbedURLsIntoFile(srcPath, dstPath, urls, seed); err != nil {
			return fmt.Errorf("嵌入文件 %s 失败: %v", relPath, err)
		}

		return nil
	})

	if err != nil {
		result.GErr(c, fmt.Errorf("批量处理失败: %v", err))
		return
	}

	// 8. 压缩输出目录
	if err := zipDirectory(outputDir, outputZipPath); err != nil {
		result.GErr(c, fmt.Errorf("压缩结果失败: %v", err))
		return
	}

	// 9. 返回结果zip文件
	resultFilename := strings.TrimSuffix(header.Filename, ".zip") + "_embedded.zip"
	c.FileAttachment(outputZipPath, resultFilename)
}

// unzipFile 解压zip文件到指定目录
func unzipFile(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		// 忽略 __MACOSX 目录及其内容
		if strings.HasPrefix(file.Name, "__MACOSX/") || strings.HasPrefix(file.Name, "__MACOSX\\") {
			continue
		}

		// 构建目标路径
		path := filepath.Join(destDir, file.Name)

		// 检查路径安全性（防止zip slip攻击）
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// 创建目录
			os.MkdirAll(path, file.Mode())
			continue
		}

		// 创建文件所在目录
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// 创建文件
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// zipDirectory 将目录压缩为zip文件
func zipDirectory(sourceDir, zipPath string) error {
	// 创建zip文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// 遍历源目录
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录本身
		if path == sourceDir {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// 忽略 __MACOSX 目录及其内容
		if strings.HasPrefix(relPath, "__MACOSX") || strings.Contains(relPath, string(os.PathSeparator)+"__MACOSX") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 创建zip文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 使用正斜杠作为路径分隔符（ZIP 标准）
		header.Name = filepath.ToSlash(relPath)

		// 如果是目录，确保名称以/结尾
		if info.IsDir() {
			header.Name += "/"
		} else {
			// 使用deflate压缩方法
			header.Method = zip.Deflate
		}

		// 创建文件头
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是目录，不需要写入内容
		if info.IsDir() {
			return nil
		}

		// 复制文件内容
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
