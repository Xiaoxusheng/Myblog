// 敏感字段静态加密（at-rest）：AES-256-GCM。
// 密文格式 "v1:" + base64(nonce(12) || ciphertext)，每值独立随机 nonce。
// 空串不加密；不带前缀的历史明文原样透传（读取兼容、迁移幂等），
// 存量数据由 MigratePIIEncryption 在启动时统一加密。
// 字段声明 gorm:"serializer:securetext" 即透明加解密，业务代码无感。
package model

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// cipherPrefix 密文版本前缀；解密时据此区分密文与历史明文
const cipherPrefix = "v1:"

var (
	cryptoMu sync.RWMutex
	cryptoAE cipher.AEAD
)

// SetCryptoKey 注入 AES-256 密钥（必须 32 字节；nil 关闭加密，字段明文存取）。
// 由 model.Open 按 Config.CryptoKey 统一调用，业务代码不要直接使用。
func SetCryptoKey(key []byte) error {
	cryptoMu.Lock()
	defer cryptoMu.Unlock()
	if len(key) == 0 {
		cryptoAE = nil
		return nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	cryptoAE = aead
	return nil
}

func currentAEAD() cipher.AEAD {
	cryptoMu.RLock()
	defer cryptoMu.RUnlock()
	return cryptoAE
}

// secureTextSerializer GORM 自定义序列化器
type secureTextSerializer struct{}

func (secureTextSerializer) Scan(_ context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	var stored string
	switch v := dbValue.(type) {
	case nil:
		stored = ""
	case string:
		stored = v
	case []byte:
		stored = string(v)
	default:
		return fmt.Errorf("securetext: 字段 %s 不支持的数据库类型 %T", field.Name, dbValue)
	}
	plain, err := decryptField(stored)
	if err != nil {
		return fmt.Errorf("securetext: 字段 %s: %w", field.Name, err)
	}
	return field.Set(context.Background(), dst, plain)
}

func (secureTextSerializer) Value(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue any) (any, error) {
	s, ok := fieldValue.(string)
	if !ok {
		return fieldValue, nil
	}
	return encryptField(s)
}

func init() {
	schema.RegisterSerializer("securetext", secureTextSerializer{})
}

// encryptField 空串原样返回；未配置密钥时透传明文（未启用加密的降级路径）
func encryptField(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	aead := currentAEAD()
	if aead == nil {
		return plain, nil
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成 nonce: %w", err)
	}
	out := aead.Seal(nonce, nonce, []byte(plain), nil)
	return cipherPrefix + base64.StdEncoding.EncodeToString(out), nil
}

func decryptField(stored string) (string, error) {
	if stored == "" || !strings.HasPrefix(stored, cipherPrefix) {
		return stored, nil // 空值 / 历史明文
	}
	aead := currentAEAD()
	if aead == nil {
		return "", errors.New("数据已加密但未配置加密密钥（BLOG_CRYPTO_KEY）")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, cipherPrefix))
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}
	if len(raw) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("密文长度不合法")
	}
	nonce, ct := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败（密钥不匹配或数据损坏）: %w", err)
	}
	return string(plain), nil
}

// MigratePIIEncryption 将存量明文敏感字段（评论 email/ip、用户 email）加密为密文。
// 幂等：仅处理不带 v1: 前缀的非空值；启动时调用，密钥未配置时为空操作。
// 读写均绕过 GORM serializer（Raw/Exec 不走字段序列化），直接操作原始列值。
func MigratePIIEncryption(db *gorm.DB) error {
	if currentAEAD() == nil {
		return nil
	}
	for _, spec := range []struct{ table, column string }{
		{"comments", "email"},
		{"comments", "ip"},
		{"users", "email"},
	} {
		type row struct {
			ID  uint
			Val string
		}
		var rows []row
		if err := db.Table(spec.table).
			Select("id", spec.column+" AS val").
			Where("`"+spec.column+"` <> '' AND `"+spec.column+"` NOT LIKE ?", cipherPrefix+"%").
			Scan(&rows).Error; err != nil {
			return fmt.Errorf("扫描 %s.%s: %w", spec.table, spec.column, err)
		}
		for _, r := range rows {
			enc, err := encryptField(r.Val)
			if err != nil {
				return fmt.Errorf("加密 %s.%s(id=%d): %w", spec.table, spec.column, r.ID, err)
			}
			if err := db.Exec("UPDATE `"+spec.table+"` SET `"+spec.column+"` = ? WHERE id = ?",
				enc, r.ID).Error; err != nil {
				return fmt.Errorf("写回 %s.%s(id=%d): %w", spec.table, spec.column, r.ID, err)
			}
		}
		if len(rows) > 0 {
			log.Printf("[migrate] 已加密存量敏感字段 %s.%s：%d 行", spec.table, spec.column, len(rows))
		}
	}
	return nil
}
