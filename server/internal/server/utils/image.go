package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// WebLogoSize 定义 tsdd-web 需要的图标尺寸
type WebLogoSize struct {
	Name   string
	Width  int
	Height int
}

// WebLogoSizes tsdd-web 容器中需要替换的所有图标文件
var WebLogoSizes = []WebLogoSize{
	{"logo.png", 1024, 1024},
	{"favicon-16x16.png", 16, 16},
	{"favicon-32x32.png", 32, 32},
	{"apple-touch-icon.png", 180, 180},
	{"icon-192x192.png", 192, 192},
	{"icon-384x384.png", 384, 384},
	{"icon-512x512.png", 512, 512},
	{"mstile-150x150.png", 150, 150},
}

// ResizePNG 使用 CatmullRom 算法高质量缩放图片
func ResizePNG(src image.Image, width, height int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// GenerateWebLogoFiles 从原始 logo 生成 tsdd-web 所需的所有尺寸文件
// 返回 map[filename]pngBytes
func GenerateWebLogoFiles(srcPath string) (map[string][]byte, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("打开logo文件失败: %v", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %v", err)
	}

	result := make(map[string][]byte, len(WebLogoSizes)+1)

	// 生成各尺寸 PNG
	for _, size := range WebLogoSizes {
		resized := ResizePNG(src, size.Width, size.Height)
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err != nil {
			return nil, fmt.Errorf("编码 %s 失败: %v", size.Name, err)
		}
		result[size.Name] = buf.Bytes()
	}

	// 生成 favicon.ico（嵌入 16x16 和 32x32 PNG）
	ico, err := generateFaviconICO(result["favicon-16x16.png"], result["favicon-32x32.png"])
	if err != nil {
		return nil, fmt.Errorf("生成favicon.ico失败: %v", err)
	}
	result["favicon.ico"] = ico

	return result, nil
}

// generateFaviconICO 生成包含多个 PNG 图片的 ICO 文件
// ICO 格式: 6字节头 + N*16字节目录 + PNG数据
func generateFaviconICO(pngImages ...[]byte) ([]byte, error) {
	var buf bytes.Buffer

	count := len(pngImages)
	headerSize := 6
	dirEntrySize := 16
	dataOffset := headerSize + count*dirEntrySize

	// ICO Header: reserved(2) + type(2) + count(2)
	binary.Write(&buf, binary.LittleEndian, uint16(0))     // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))     // type: 1=ICO
	binary.Write(&buf, binary.LittleEndian, uint16(count)) // image count

	// 计算每个图片的偏移
	offset := dataOffset
	sizes := []int{16, 32} // 对应传入的 PNG 图片尺寸
	for i, pngData := range pngImages {
		size := byte(0) // 0 表示 256
		if i < len(sizes) && sizes[i] < 256 {
			size = byte(sizes[i])
		}
		// Directory entry: width(1) + height(1) + colorCount(1) + reserved(1) +
		//                  planes(2) + bpp(2) + dataSize(4) + offset(4)
		buf.WriteByte(size)                                                   // width
		buf.WriteByte(size)                                                   // height
		buf.WriteByte(0)                                                      // color count (0=no palette)
		buf.WriteByte(0)                                                      // reserved
		binary.Write(&buf, binary.LittleEndian, uint16(1))                    // color planes
		binary.Write(&buf, binary.LittleEndian, uint16(32))                   // bits per pixel
		binary.Write(&buf, binary.LittleEndian, uint32(len(pngData)))         // data size
		binary.Write(&buf, binary.LittleEndian, uint32(offset))              // data offset
		offset += len(pngData)
	}

	// 写入 PNG 数据
	for _, pngData := range pngImages {
		buf.Write(pngData)
	}

	return buf.Bytes(), nil
}
