package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Content is the JSON structure for content.json
type Content struct {
	UpdatedAt        string         `json:"updated_at"`
	Versions         []VersionEntry `json:"versions"`
	UserAgreement    string         `json:"user_agreement"`
	PrivacyPolicy    string         `json:"privacy_policy"`
	UserAgreementPre string         `json:"user_agreement_pre"`
	PrivacyPolicyPre string         `json:"privacy_policy_pre"`
}

// VersionEntry is the fixed structure for items in versions
type VersionEntry struct {
	Channel string `json:"channel"`
	Version string `json:"version"`
}

// GenerateContentJSON embeds HTMLs into content.json and writes compact JSON (no spaces)
// - termsHTMLPath: path to terms.html to embed into user_agreement
// - privacyHTMLPath: optional path to privacy.html to embed into privacy_policy (empty to skip)
// - termsPreHTMLPath: optional path to pre terms html to embed into user_agreement_pre (empty to skip)
// - privacyPreHTMLPath: optional path to pre privacy html to embed into privacy_policy_pre (empty to skip)
// - outJSONPath: destination JSON path
// Values for updated_at and versions are hardcoded per current requirements.
func GenerateContentJSON(termsHTMLPath, privacyHTMLPath, termsPreHTMLPath, privacyPreHTMLPath, outJSONPath string) error {
	c := Content{
		UpdatedAt: "2025-09-15 10:00:00",
		Versions: []VersionEntry{
			{Channel: "oppo", Version: "1.0.0"},
			{Channel: "vivo", Version: "1.0.0"},
			{Channel: "xiaomi", Version: "1.0.0"},
			{Channel: "rongyao", Version: "1.0.0"},
			{Channel: "huawei", Version: "1.0.0"},
			{Channel: "local", Version: "1.0.0"},
		},
	}
	return generateContentJSONInternal(c, termsHTMLPath, privacyHTMLPath, termsPreHTMLPath, privacyPreHTMLPath, outJSONPath)
}

// GenerateContentJSONFromData 使用自定义数据生成content.json
func GenerateContentJSONFromData(updatedAt string, versions []VersionEntry, termsHTMLPath, privacyHTMLPath, termsPreHTMLPath, privacyPreHTMLPath, outJSONPath string) error {
	c := Content{
		UpdatedAt: updatedAt,
		Versions:  versions,
	}
	return generateContentJSONInternal(c, termsHTMLPath, privacyHTMLPath, termsPreHTMLPath, privacyPreHTMLPath, outJSONPath)
}

// generateContentJSONInternal 内部实现：生成content.json
func generateContentJSONInternal(c Content, termsHTMLPath, privacyHTMLPath, termsPreHTMLPath, privacyPreHTMLPath, outJSONPath string) error {
	terms, err := os.ReadFile(termsHTMLPath)
	if err != nil {
		return err
	}
	c.UserAgreement = compressHTMLWhitespace(string(terms))

	if privacyHTMLPath != "" {
		if pp, err := os.ReadFile(privacyHTMLPath); err == nil {
			c.PrivacyPolicy = compressHTMLWhitespace(string(pp))
		}
	}

	if termsPreHTMLPath != "" {
		if tp, err := os.ReadFile(termsPreHTMLPath); err == nil {
			c.UserAgreementPre = compressHTMLWhitespace(string(tp))
		}
	}

	if privacyPreHTMLPath != "" {
		if pp2, err := os.ReadFile(privacyPreHTMLPath); err == nil {
			c.PrivacyPolicyPre = compressHTMLWhitespace(string(pp2))
		}
	}

	out, err := json.Marshal(c) // compact JSON without spaces
	if err != nil {
		return err
	}
	return os.WriteFile(outJSONPath, out, 0644)
}

var (
	reBetweenTags = regexp.MustCompile(`>\s+<`)
	reMultiSpace  = regexp.MustCompile(`\s+`)
)

// compressHTMLWhitespace reduces unnecessary whitespace in HTML:
// - normalizes line breaks/tabs
// - collapses consecutive whitespace to a single space
// - removes whitespace between tags
// - trims leading/trailing whitespace
func compressHTMLWhitespace(in string) string {
	s := strings.ReplaceAll(in, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", " ")
	s = reMultiSpace.ReplaceAllString(s, " ")
	s = reBetweenTags.ReplaceAllString(s, "><")
	s = strings.TrimSpace(s)
	return s
}

const gcmNonceSize = 12

var errInvalidKeyLength = errors.New("key must be 32 bytes (AES-256)")

func ensureAES256Key(key []byte) error {
	if len(key) != 32 {
		return errInvalidKeyLength
	}
	return nil
}

// EncryptFile reads JSON from inputPath, encrypts using AES-256-GCM, and writes out as: nonce(12B) + ciphertext||tag
func EncryptFile(inputPath, outputPath string, key []byte) error {
	if err := ensureAES256Key(key); err != nil {
		return err
	}

	plain, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := aead.Seal(nil, nonce, plain, nil)

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.Write(nonce); err != nil {
		return err
	}
	if _, err := out.Write(ciphertext); err != nil {
		return err
	}
	return out.Sync()
}

// DecryptFile reads encrypted file (nonce(12B) + ciphertext||tag), decrypts with AES-256-GCM, and writes JSON to outputPath
func DecryptFile(inputPath, outputPath string, key []byte) error {
	if err := ensureAES256Key(key); err != nil {
		return err
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	if len(data) < gcmNonceSize+16 {
		return errors.New("ciphertext too short")
	}

	nonce := data[:gcmNonceSize]
	ciphertext := data[gcmNonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, plain, 0644); err != nil {
		return err
	}
	return nil
}

// ParseHexKey decodes a 64-hex-character string into a 32-byte AES-256 key
func ParseHexKey(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errInvalidKeyLength
	}
	return key, nil
}

// GenerateRandomHexKey creates a random 32-byte key and returns a 64-hex string
func GenerateRandomHexKey() string {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	return hex.EncodeToString(key)
}

func _main() {
	// 生成文件
	GenerateContentJSON("terms.html", "privacy.html", "terms_pre.html", "privacy_pre.html", "content.json")

	// 加密文件
	secret := "3d4270b340f381cfd70b8bed30c3191845e52a45c9aec15e3e83de4500761af4" // GenerateRandomHexKey()
	fmt.Println("secret:", secret)
	key, _ := ParseHexKey(secret)
	_ = EncryptFile("content.json", "content.txt", key)
	_ = DecryptFile("content.txt", "content.dec.json", key)
}
