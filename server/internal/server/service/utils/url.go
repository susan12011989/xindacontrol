package utils

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// -------------------- Config & Constants --------------------
var URL_MAGIC = []byte("URLS")

// 文件尾部 Footer MAGIC（用于从文件结尾定位载荷）
var URL_FILE_FOOTER_MAGIC = []byte("URLF")

const URL_VERSION byte = 1

// -------------------- Serialization --------------------
func packURLs(urls []string) ([]byte, error) {
	var entries bytes.Buffer
	for _, s := range urls {
		b := []byte(s)
		if len(b) > 0xFFFF {
			return nil, fmt.Errorf("url too long: %d bytes", len(b))
		}
		// length (uint16 LE) + bytes
		binary.Write(&entries, binary.LittleEndian, uint16(len(b)))
		entries.Write(b)
	}
	var out bytes.Buffer
	out.Write(URL_MAGIC)
	out.WriteByte(URL_VERSION)
	binary.Write(&out, binary.LittleEndian, uint16(len(urls)))
	out.Write(entries.Bytes())

	// CRC over version..last entry byte (skip magic)
	crcRegion := out.Bytes()[4:]
	crc := crc32.ChecksumIEEE(crcRegion)
	binary.Write(&out, binary.LittleEndian, crc)
	return out.Bytes(), nil
}

func unpackURLs(payload []byte) ([]string, error) {
	if len(payload) < 4+1+2+4 {
		return nil, errors.New("payload too short")
	}
	if !bytes.Equal(payload[:4], URL_MAGIC) {
		return nil, errors.New("magic mismatch")
	}
	p := 4
	version := payload[p]
	p++
	if version != URL_VERSION {
		return nil, fmt.Errorf("unsupported version %d", version)
	}
	count := int(binary.LittleEndian.Uint16(payload[p : p+2]))
	p += 2
	urls := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if p+2 > len(payload) {
			return nil, errors.New("payload truncated when reading url length")
		}
		L := int(binary.LittleEndian.Uint16(payload[p : p+2]))
		p += 2
		if p+L > len(payload) {
			return nil, errors.New("payload truncated when reading url bytes")
		}
		u := string(payload[p : p+L])
		p += L
		urls = append(urls, u)
	}
	// CRC check
	if p+4 > len(payload) {
		return nil, errors.New("payload missing CRC")
	}
	given := binary.LittleEndian.Uint32(payload[p : p+4])
	calc := crc32.ChecksumIEEE(payload[4:p]) // version..last entry
	if given != calc {
		return nil, fmt.Errorf("crc mismatch: given %08x calc %08x", given, calc)
	}
	return urls, nil
}

// -------------------- PRNG (xorshift32) --------------------
type URLXorShift32 struct {
	state uint32
}

func NewURLXorshift(seed uint32) *URLXorShift32 {
	if seed == 0 {
		seed = 0xA5A5A5A5 // avoid zero-state
	}
	return &URLXorShift32{state: seed}
}

func (x *URLXorShift32) Next() uint32 {
	s := x.state
	s ^= s << 13
	s ^= s >> 17
	s ^= s << 5
	x.state = s
	return s
}

// xorBufferWithXorshift 使用 xorshift32 与给定 seed 对缓冲区进行就地 XOR 混淆/还原
func xorBufferWithURLXorshift(buf []byte, seed uint32) {
	rng := NewURLXorshift(seed)
	for i := 0; i < len(buf); i++ {
		k := byte(rng.Next() & 0xFF)
		buf[i] ^= k
	}
}

// -------------------- Generic File Tail Embed & Extract --------------------
// Footer 结构（从文件末尾回溯读取）：
//  - FILE_FOOTER_MAGIC (4 bytes: "URLF")
//  - VERSION (1 byte)
//  - payloadLen (4 bytes, LE)
//  - payloadCRC (4 bytes, LE)  // 针对解密后的 payload 校验

// EmbedURLsIntoFile 将 URL 列表以尾部载荷的方式嵌入到任意文件中
// seed 为固定的 444013（调用方可传入固定值），内部采用 xorshift32 对载荷进行 XOR 混淆
func EmbedURLsIntoFile(srcPath, dstPath string, urls []string, seed uint32) error {
	// 读取原文件
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	// 序列化 URL 列表
	payload, err := packURLs(urls)
	if err != nil {
		return err
	}
	// 计算 payload CRC（针对解密后明文）
	payloadCRC := crc32.ChecksumIEEE(payload)

	// 混淆写入副本
	payloadEnc := make([]byte, len(payload))
	copy(payloadEnc, payload)
	xorBufferWithURLXorshift(payloadEnc, seed)

	// 构造 Footer
	var footer bytes.Buffer
	footer.Write(URL_FILE_FOOTER_MAGIC)
	footer.WriteByte(URL_VERSION)
	binary.Write(&footer, binary.LittleEndian, uint32(len(payloadEnc)))
	binary.Write(&footer, binary.LittleEndian, payloadCRC)

	// 写出：原文件 + 加密载荷 + Footer
	out := bytes.NewBuffer(nil)
	out.Grow(len(srcBytes) + len(payloadEnc) + footer.Len())
	out.Write(srcBytes)
	out.Write(payloadEnc)
	out.Write(footer.Bytes())
	return os.WriteFile(dstPath, out.Bytes(), 0644)
}

// ExtractURLsFromFile 从文件尾部提取之前嵌入的 URL 列表
func ExtractURLsFromFile(srcPath string, seed uint32) ([]string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	// Footer 固定长度：4(MAGIC)+1(V)+4(len)+4(CRC)=13
	const footerLen = 13
	if size < footerLen {
		return nil, errors.New("file too small for footer")
	}
	// 读取 Footer
	if _, err := f.Seek(size-footerLen, io.SeekStart); err != nil {
		return nil, err
	}
	foot := make([]byte, footerLen)
	if _, err := io.ReadFull(f, foot); err != nil {
		return nil, err
	}
	// 校验 Footer
	if !bytes.Equal(foot[:4], URL_FILE_FOOTER_MAGIC) {
		return nil, errors.New("footer magic not found: no embedded payload")
	}
	version := foot[4]
	if version != URL_VERSION {
		return nil, fmt.Errorf("unsupported version %d", version)
	}
	payloadLen := binary.LittleEndian.Uint32(foot[5:9])
	payloadCRC := binary.LittleEndian.Uint32(foot[9:13])

	// 边界检查
	if int64(payloadLen)+int64(footerLen) > size {
		return nil, errors.New("invalid footer: payload length exceeds file size")
	}

	// 读取加密载荷
	if _, err := f.Seek(size-int64(footerLen)-int64(payloadLen), io.SeekStart); err != nil {
		return nil, err
	}
	payloadEnc := make([]byte, payloadLen)
	if _, err := io.ReadFull(f, payloadEnc); err != nil {
		return nil, err
	}
	// 解密（XOR 还原）
	xorBufferWithURLXorshift(payloadEnc, seed)
	// 校验 CRC
	calc := crc32.ChecksumIEEE(payloadEnc)
	if calc != payloadCRC {
		return nil, fmt.Errorf("payload crc mismatch: given %08x calc %08x", payloadCRC, calc)
	}
	// 解析 URL 列表
	urls, err := unpackURLs(payloadEnc)
	if err != nil {
		return nil, err
	}
	return urls, nil
}

// -------------------- CLI --------------------
func main() {
	mode := flag.String("mode", "embed", "embed or extract")
	in := flag.String("in", "./test.mp4", "input file path")
	out := flag.String("out", "./test_out.mp4", "output file path (for embed-file)")
	seed := flag.Uint("seed", 444013, "seed (uint32) for PRNG")
	urlfile := flag.String("urlfile", "./url.txt", "file with URLs (one per line) for embed-file")
	flag.Parse()

	switch *mode {
	case "embed":
		if *in == "" || *out == "" {
			fmt.Println("embed-file mode requires -in and -out")
			return
		}
		if *urlfile == "" {
			fmt.Println("no URLs provided; use -urlfile")
			return
		}
		f, err := os.Open(*urlfile)
		if err != nil {
			fmt.Println("read urlfile err:", err)
			return
		}
		defer f.Close()
		scanner := bufio.NewReader(f)
		urls := make([]string, 0, 8)
		for {
			line, _, err := scanner.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Println("read urlfile err:", err)
				return
			}
			urls = append(urls, string(line))
		}
		if len(urls) == 0 {
			fmt.Println("no URLs provided; use -urlfile")
			return
		}
		if err := EmbedURLsIntoFile(*in, *out, urls, uint32(*seed)); err != nil {
			fmt.Println("embed-file error:", err)
			return
		}
		fmt.Println("embed-file success ->", *out)
	case "extract":
		if *in == "" {
			fmt.Println("extract-file mode requires -in")
			return
		}
		urls, err := ExtractURLsFromFile(*in, uint32(*seed))
		if err != nil {
			fmt.Println("extract-file error:", err)
			return
		}
		fmt.Println("extracted URLs:")
		for _, u := range urls {
			fmt.Println(" -", u)
		}
	default:
		fmt.Println("unknown mode:", *mode)
	}
}
