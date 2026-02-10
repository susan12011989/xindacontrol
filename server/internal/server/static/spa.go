package static

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// NewSPAHandler 返回一个带 SPA 回退能力的处理器（不使用 FileServer，避免 301 重定向）。
// - 仅对 GET/HEAD 生效
// - 资源存在：直接返回（非 HTML 强缓存）
// - 资源不存在且 Accept 包含 text/html：回退到 index.html
var hashedNamePattern = regexp.MustCompile(`-[A-Za-z0-9_-]{6,}\.`)

func NewSPAHandler(fsys fs.FS, index string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = index
		}

		serve := func(name string) {
			f, err := fsys.Open(name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}

			// 缓存策略
			if strings.HasSuffix(name, ".html") {
				// HTML 采用短期缓存，降低更新延迟风险
				w.Header().Set("Cache-Control", "public, max-age=300")
			} else if hashedNamePattern.MatchString(name) {
				// 仅对带 hash 的构建产物使用强缓存
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				// 其他静态资源给予较短缓存
				w.Header().Set("Cache-Control", "public, max-age=3600")
			}

			// Content-Type
			ct := mime.TypeByExtension(path.Ext(name))
			if ct == "" {
				ct = "application/octet-stream"
			}
			w.Header().Set("Content-Type", ct)

			// 优先使用 ServeContent 支持 Range/缓存
			if rs, ok := f.(io.ReadSeeker); ok {
				http.ServeContent(w, r, name, info.ModTime(), rs)
				return
			}

			// 回退到一次性读取
			data, err := io.ReadAll(f)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		}

		// 先尝试直接命中资源
		if _, err := fs.Stat(fsys, p); err == nil {
			serve(p)
			return
		}

		// 不存在：仅浏览器 HTML 请求才回退到 index.html
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			serve(index)
			return
		}

		http.NotFound(w, r)
	})
}
