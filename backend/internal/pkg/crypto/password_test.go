package crypto

import "testing"

// TestHashAndCompare 验证密码加密和比对的完整流程。
func TestHashAndCompare(t *testing.T) {
	password := "admin123"

	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// 正确密码应匹配
	if !ComparePassword(hashed, password) {
		t.Error("ComparePassword() 正确密码验证失败")
	}

	// 错误密码不应匹配
	if ComparePassword(hashed, "wrong") {
		t.Error("ComparePassword() 错误密码不应通过验证")
	}
}
