package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"os"
)

// -------------------- Config & Constants --------------------
var MAGIC = []byte("IPSG")

// 通用文件尾部追加模式的 Footer MAGIC（用于从任意文件结尾定位载荷）
var FILE_FOOTER_MAGIC = []byte("IPGF")

const VERSION byte = 1

// -------------------- Serialization --------------------
func packIPs(ips []string) ([]byte, error) {
	var entries bytes.Buffer
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip: %s", s)
		}
		if ip4 := ip.To4(); ip4 != nil {
			entries.WriteByte(0x04)
			entries.WriteByte(byte(4))
			entries.Write(ip4)
		} else {
			ip16 := ip.To16()
			if ip16 == nil {
				return nil, fmt.Errorf("invalid ip (not v4/v6): %s", s)
			}
			entries.WriteByte(0x06)
			entries.WriteByte(byte(16))
			entries.Write(ip16)
		}
	}
	var out bytes.Buffer
	out.Write(MAGIC)
	out.WriteByte(VERSION)
	binary.Write(&out, binary.LittleEndian, uint16(len(ips)))
	out.Write(entries.Bytes())

	// CRC over version..last addr byte (version + count + entries)
	crcRegion := out.Bytes()[4:] // skip magic
	crc := crc32.ChecksumIEEE(crcRegion)
	binary.Write(&out, binary.LittleEndian, crc)
	return out.Bytes(), nil
}

func unpackIPs(payload []byte) ([]string, error) {
	if len(payload) < 4+1+2+4 {
		return nil, errors.New("payload too short")
	}
	if !bytes.Equal(payload[:4], MAGIC) {
		return nil, errors.New("magic mismatch")
	}
	p := 4
	version := payload[p]
	p++
	if version != VERSION {
		return nil, fmt.Errorf("unsupported version %d", version)
	}
	count := binary.LittleEndian.Uint16(payload[p : p+2])
	p += 2
	ips := make([]string, 0, count)
	for i := 0; i < int(count); i++ {
		if p+2 > len(payload) {
			return nil, errors.New("payload truncated when reading entries")
		}
		typ := payload[p]
		p++
		L := int(payload[p])
		p++
		if p+L > len(payload) {
			return nil, errors.New("payload truncated reading address")
		}
		addr := payload[p : p+L]
		p += L
		if typ == 0x04 && L == 4 {
			ips = append(ips, net.IP(addr).String())
		} else if typ == 0x06 && L == 16 {
			ips = append(ips, net.IP(addr).String())
		} else {
			return nil, errors.New("bad entry type/length")
		}
	}
	// CRC check
	if p+4 > len(payload) {
		return nil, errors.New("payload missing CRC")
	}
	given := binary.LittleEndian.Uint32(payload[p : p+4])
	calc := crc32.ChecksumIEEE(payload[4:p]) // version..lastaddr
	if given != calc {
		return nil, fmt.Errorf("crc mismatch: given %08x calc %08x", given, calc)
	}
	return ips, nil
}

// -------------------- PRNG (xorshift32) & shuffle --------------------
type XorShift32 struct {
	state uint32
}

func NewXorshift(seed uint32) *XorShift32 {
	if seed == 0 {
		seed = 0xA5A5A5A5 // avoid zero-state
	}
	return &XorShift32{state: seed}
}

func (x *XorShift32) Next() uint32 {
	s := x.state
	s ^= s << 13
	s ^= s >> 17
	s ^= s << 5
	x.state = s
	return s
}

// -------------------- Generic File Tail Embed & Extract --------------------
// 方案：在任意文件尾部追加 [payloadEnc][FOOTER]。
// FOOTER 结构（从文件末尾回溯读取）：
//  - FILE_FOOTER_MAGIC (4 bytes: "IPGF")
//  - VERSION (1 byte)
//  - payloadLen (4 bytes, LE)
//  - payloadCRC (4 bytes, LE)  // 针对解密后的 payload 校验
// 提取：读取末尾 13 字节 Footer，定位前置 payloadEnc，按 seed 进行 XOR 复原，校验 CRC -> unpackIPs。

// xorBufferWithXorshift 使用 xorshift32 与给定 seed 对缓冲区进行就地 XOR 混淆/还原
func xorBufferWithXorshift(buf []byte, seed uint32) {
	rng := NewXorshift(seed)
	for i := 0; i < len(buf); i++ {
		// 使用 Next 的低 8 位作为 XOR 字节
		k := byte(rng.Next() & 0xFF)
		buf[i] ^= k
	}
}

func EmbedIntoFile(srcPath, dstPath string, ips []string, seed uint32) error {
	// 读取原文件
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	// 序列化 IP 列表
	payload, err := packIPs(ips)
	if err != nil {
		return err
	}
	// 计算 payload CRC（针对解密后明文）
	payloadCRC := crc32.ChecksumIEEE(payload)

	// 混淆写入副本
	payloadEnc := make([]byte, len(payload))
	copy(payloadEnc, payload)
	xorBufferWithXorshift(payloadEnc, seed)

	// 构造 Footer
	var footer bytes.Buffer
	footer.Write(FILE_FOOTER_MAGIC)
	footer.WriteByte(VERSION)
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

// EmbedIPsToBytes 将IP嵌入到字节数组中（内存版本，用于上传到云存储）
func EmbedIPsToBytes(srcBytes []byte, ips []string, seed uint32) ([]byte, error) {
	// 序列化 IP 列表
	payload, err := packIPs(ips)
	if err != nil {
		return nil, err
	}
	// 计算 payload CRC（针对解密后明文）
	payloadCRC := crc32.ChecksumIEEE(payload)

	// 混淆写入副本
	payloadEnc := make([]byte, len(payload))
	copy(payloadEnc, payload)
	xorBufferWithXorshift(payloadEnc, seed)

	// 构造 Footer
	var footer bytes.Buffer
	footer.Write(FILE_FOOTER_MAGIC)
	footer.WriteByte(VERSION)
	binary.Write(&footer, binary.LittleEndian, uint32(len(payloadEnc)))
	binary.Write(&footer, binary.LittleEndian, payloadCRC)

	// 写出：原文件 + 加密载荷 + Footer
	out := bytes.NewBuffer(nil)
	out.Grow(len(srcBytes) + len(payloadEnc) + footer.Len())
	out.Write(srcBytes)
	out.Write(payloadEnc)
	out.Write(footer.Bytes())
	return out.Bytes(), nil
}

func ExtractFromFile(srcPath string, seed uint32) ([]string, error) {
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
	if !bytes.Equal(foot[:4], FILE_FOOTER_MAGIC) {
		return nil, errors.New("footer magic not found: no embedded payload")
	}
	version := foot[4]
	if version != VERSION {
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
	xorBufferWithXorshift(payloadEnc, seed)
	// 校验 CRC
	calc := crc32.ChecksumIEEE(payloadEnc)
	if calc != payloadCRC {
		return nil, fmt.Errorf("payload crc mismatch: given %08x calc %08x", payloadCRC, calc)
	}
	// 解析 IP 列表（内层还有 packIPs 的 MAGIC/CRC 校验）
	ips, err := unpackIPs(payloadEnc)
	if err != nil {
		return nil, err
	}
	return ips, nil
}
