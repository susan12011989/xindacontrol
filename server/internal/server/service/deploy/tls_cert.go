package deploy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ========== 证书生成 ==========

// GenerateTlsCerts 为指定商户生成 CA 根证书 + 服务器证书，存入数据库
func GenerateTlsCerts(merchantId, validityDays int) (*model.GenerateTlsCertResp, error) {
	if validityDays <= 0 {
		validityDays = 3650 // 默认10年
	}

	// 检查该商户是否已有证书
	var existing entity.TlsCertificates
	has, err := dbs.DBAdmin.Where("name = ? AND merchant_id = ? AND status = 1", "gost-ca", merchantId).Get(&existing)
	if err != nil {
		return nil, fmt.Errorf("查询证书失败: %v", err)
	}
	if has {
		return nil, errors.New("该商户已存在有效的 CA 证书，如需重新生成请先停用旧证书")
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(validityDays) * 24 * time.Hour)

	// 1. 生成 CA 根证书
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成 CA 密钥失败: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "TSDD GOST CA",
			Organization: []string{"TSDD"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("生成 CA 证书失败: %v", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	caKeyDER, _ := x509.MarshalECPrivateKey(caKey)
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: caKeyDER})
	caFingerprint := sha256Fingerprint(caCertDER)

	// 2. 生成服务器证书（CA 签发）
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成服务器密钥失败: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   "TSDD GOST Server",
			Organization: []string{"TSDD"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("生成服务器证书失败: %v", err)
	}

	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	serverKeyDER, _ := x509.MarshalECPrivateKey(serverKey)
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyDER})
	serverFingerprint := sha256Fingerprint(serverCertDER)

	// 3. 存入数据库
	now := time.Now()
	caCert := &entity.TlsCertificates{
		MerchantId:  merchantId,
		Name:        "gost-ca",
		CertType:    1,
		CertPem:     string(caCertPEM),
		KeyPem:      string(caKeyPEM),
		Fingerprint: caFingerprint,
		ExpiresAt:   notAfter,
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	serverCert := &entity.TlsCertificates{
		MerchantId:  merchantId,
		Name:        "gost-server",
		CertType:    2,
		CertPem:     string(serverCertPEM),
		KeyPem:      string(serverKeyPEM),
		Fingerprint: serverFingerprint,
		ExpiresAt:   notAfter,
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = dbs.DBAdmin.Insert(caCert)
	if err != nil {
		return nil, fmt.Errorf("保存 CA 证书失败: %v", err)
	}
	_, err = dbs.DBAdmin.Insert(serverCert)
	if err != nil {
		return nil, fmt.Errorf("保存服务器证书失败: %v", err)
	}

	logx.Infof("TLS 证书生成成功: 商户ID=%d, CA(id=%d) Server(id=%d), 有效期至 %s",
		merchantId, caCert.Id, serverCert.Id, notAfter.Format("2006-01-02"))

	return &model.GenerateTlsCertResp{
		CA:     certToResp(caCert),
		Server: certToResp(serverCert),
	}, nil
}

// GetTlsCerts 获取指定商户当前有效的证书
func GetTlsCerts(merchantId int) (*model.GenerateTlsCertResp, error) {
	var certs []entity.TlsCertificates
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).OrderBy("cert_type ASC").Find(&certs)
	if err != nil {
		return nil, fmt.Errorf("查询证书失败: %v", err)
	}

	resp := &model.GenerateTlsCertResp{}
	for _, c := range certs {
		item := certToResp(&c)
		if c.CertType == 1 {
			resp.CA = item
		} else if c.CertType == 2 {
			resp.Server = item
		}
	}

	return resp, nil
}

// DisableTlsCerts 停用指定商户的证书（允许重新生成）
func DisableTlsCerts(merchantId int) error {
	_, err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Cols("status", "updated_at").Update(&entity.TlsCertificates{
		Status:    0,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("停用证书失败: %v", err)
	}
	logx.Infof("TLS 证书已停用: 商户ID=%d", merchantId)
	return nil
}

// GetCertFingerprint 获取指定商户的证书指纹（供 App 端 Pinning）
func GetCertFingerprint(merchantId int) (map[string]string, error) {
	var certs []entity.TlsCertificates
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).Find(&certs)
	if err != nil {
		return nil, fmt.Errorf("查询证书失败: %v", err)
	}

	result := make(map[string]string)
	for _, c := range certs {
		result[c.Name] = c.Fingerprint
	}
	return result, nil
}

// ========== 辅助函数 ==========

func sha256Fingerprint(certDER []byte) string {
	hash := sha256.Sum256(certDER)
	parts := make([]string, len(hash))
	for i, b := range hash {
		parts[i] = hex.EncodeToString([]byte{b})
	}
	return strings.ToUpper(strings.Join(parts, ":"))
}

func certToResp(c *entity.TlsCertificates) model.TlsCertificateResp {
	return model.TlsCertificateResp{
		Id:          c.Id,
		MerchantId:  c.MerchantId,
		Name:        c.Name,
		CertType:    c.CertType,
		Fingerprint: c.Fingerprint,
		ExpiresAt:   c.ExpiresAt.Format("2006-01-02 15:04:05"),
		Status:      c.Status,
		CreatedAt:   c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
