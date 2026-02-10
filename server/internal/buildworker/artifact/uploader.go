package artifact

import (
	"bytes"
	"fmt"
	"io"
	"server/internal/buildworker/cfg"
	"server/internal/server/cloud/aliyun"
	"server/internal/server/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type Uploader struct {
	storageType    string
	cloudAccountID int64
	regionID       string
	bucket         string
	objectPrefix   string
}

func NewUploader() *Uploader {
	return &Uploader{
		storageType:    cfg.C.Storage.Type,
		cloudAccountID: cfg.C.Storage.CloudAccountID,
		regionID:       cfg.C.Storage.RegionID,
		bucket:         cfg.C.Storage.Bucket,
		objectPrefix:   cfg.C.Storage.ObjectPrefix,
	}
}

// UploadFromSSH 从 SSH 服务器上传文件到云存储
func (u *Uploader) UploadFromSSH(sshClient *utils.SSHClient, remotePath, objectKey string) (string, error) {
	// 读取远程文件内容
	catCmd := fmt.Sprintf("cat '%s'", remotePath)

	session, err := sshClient.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := session.Start(catCmd); err != nil {
		return "", err
	}

	content, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	session.Wait()

	if len(content) == 0 {
		return "", fmt.Errorf("file is empty: %s", remotePath)
	}

	return u.UploadContent(content, objectKey, "")
}

// UploadContent 上传内容到云存储
func (u *Uploader) UploadContent(content []byte, objectKey, contentType string) (string, error) {
	fullKey := u.objectPrefix + objectKey
	reader := bytes.NewReader(content)

	switch u.storageType {
	case "aliyun":
		return u.uploadToAliyun(reader, fullKey)
	case "tencent":
		return u.uploadToTencent(reader, fullKey)
	case "aws":
		return u.uploadToAWS(reader, fullKey)
	case "local":
		return u.uploadToLocal(content, fullKey)
	default:
		return "", fmt.Errorf("unsupported storage type: %s", u.storageType)
	}
}

func (u *Uploader) uploadToAliyun(reader io.Reader, objectKey string) (string, error) {
	bucket, err := aliyun.GetOssBucket(0, u.cloudAccountID, u.regionID, u.bucket)
	if err != nil {
		return "", fmt.Errorf("get oss bucket failed: %w", err)
	}

	if err := bucket.PutObject(objectKey, reader); err != nil {
		return "", fmt.Errorf("put object failed: %w", err)
	}

	// 返回访问 URL
	url := fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s",
		u.bucket, u.regionID, objectKey)

	logx.Infof("Uploaded to Aliyun OSS: %s", url)
	return url, nil
}

func (u *Uploader) uploadToTencent(reader io.Reader, objectKey string) (string, error) {
	// 腾讯云 COS 上传
	// TODO: 根据需要实现腾讯云 COS 上传
	return "", fmt.Errorf("tencent COS upload not implemented yet")
}

func (u *Uploader) uploadToAWS(reader io.Reader, objectKey string) (string, error) {
	// AWS S3 上传
	// TODO: 根据需要实现 AWS S3 上传
	return "", fmt.Errorf("AWS S3 upload not implemented yet")
}

func (u *Uploader) uploadToLocal(content []byte, objectKey string) (string, error) {
	// 本地存储（用于开发测试）
	// TODO: 实现本地文件存储
	return "", fmt.Errorf("local storage not implemented yet")
}

// GetDownloadURL 获取下载URL（带签名）
func (u *Uploader) GetDownloadURL(objectKey string, expireSeconds int64) (string, error) {
	fullKey := u.objectPrefix + objectKey

	switch u.storageType {
	case "aliyun":
		bucket, err := aliyun.GetOssBucket(0, u.cloudAccountID, u.regionID, u.bucket)
		if err != nil {
			return "", err
		}
		return bucket.SignURL(fullKey, "GET", expireSeconds)
	default:
		return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s",
			u.bucket, u.regionID, fullKey), nil
	}
}
