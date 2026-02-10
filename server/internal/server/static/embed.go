package static

import (
	"embed"
	"io/fs"
	"net/http"
)

// 嵌入前端静态文件
// 关于 go:embed：指令必须在 go:embed 和变量声明之间没有空行
//
//go:embed all:dist
var staticFiles embed.FS

// GetFileSystem 返回嵌入的静态文件系统
// 会自动去除 dist 前缀，使得访问路径更简洁
func GetFileSystem() http.FileSystem {
	// 创建子文件系统，去除 dist 前缀
	fsys, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// Exists 检查文件是否存在
func Exists(path string) bool {
	fsys, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		return false
	}
	_, err = fsys.Open(path)
	return err == nil
}

// ReadFile 读取文件内容
func ReadFile(path string) ([]byte, error) {
	return staticFiles.ReadFile("dist/" + path)
}

// FS 返回去掉 dist 前缀后的只读文件系统（用于 http.FS / fs.Stat 等）
func FS() fs.FS {
	fsys, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic(err)
	}
	return fsys
}
