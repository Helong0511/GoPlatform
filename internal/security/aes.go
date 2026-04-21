package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// AESUtils AES-256-GCM 加密工具
 type AESUtils struct {
	Key []byte
}

// NewAESUtils 创建AES工具实例
func NewAESUtils(keyBase64 string) (*AESUtils, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	
	// 验证密钥长度是否为AES标准长度（16/24/32字节）
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, errors.New("invalid AES key length: must be 16, 24, or 32 bytes")
	}
	
	return &AESUtils{Key: key}, nil
}

// Encrypt 加密数据
func (a *AESUtils) Encrypt(plaintext string) (string, error) {
	// 创建AES加密器
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机IV
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, iv, []byte(plaintext), nil)

	// 组合IV和密文
	encrypted := append(iv, ciphertext...)

	// 返回十六进制字符串
	return hex.EncodeToString(encrypted), nil
}

// Decrypt 解密数据
func (a *AESUtils) Decrypt(encryptedHex string) (string, error) {
	// 解码十六进制字符串
	encrypted, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", err
	}

	// 创建AES解密器
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 检查数据长度
	if len(encrypted) < gcm.NonceSize() {
		return "", errors.New("加密数据格式错误")
	}

	// 分离IV和密文
	iv, ciphertext := encrypted[:gcm.NonceSize()], encrypted[gcm.NonceSize():]

	// 解密数据
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
