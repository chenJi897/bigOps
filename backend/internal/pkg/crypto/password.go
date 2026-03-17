// Package crypto 提供密码加密与验证功能。
// 使用 bcrypt 算法对用户密码进行单向哈希，保障存储安全。
package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

// bcrypt 计算成本，值越大越安全但越慢，10 是推荐的平衡值。
const defaultCost = 10

// HashPassword 对明文密码进行 bcrypt 加密，返回哈希字符串。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), defaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword 比较哈希密码和明文密码是否匹配。
// 匹配返回 true，不匹配或出错返回 false。
func ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
