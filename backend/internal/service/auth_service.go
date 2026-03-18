// Package service 提供业务逻辑层，处理认证、用户管理等核心业务。
package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/crypto"
	"github.com/bigops/platform/internal/pkg/database"
	jwtPkg "github.com/bigops/platform/internal/pkg/jwt"
	"github.com/bigops/platform/internal/repository"
)

// AuthService 认证服务，处理注册、登录、登出等认证相关业务。
type AuthService struct {
	userRepo *repository.UserRepository
}

// NewAuthService 创建 AuthService 实例。
func NewAuthService() *AuthService {
	return &AuthService{
		userRepo: repository.NewUserRepository(),
	}
}

// Register 用户注册：校验密码复杂度 → 校验唯一性 → 密码加密 → 写入数据库。
func (s *AuthService) Register(username, password, email string) error {
	// 密码复杂度校验
	if err := checkPasswordComplexity(password); err != nil {
		return err
	}

	// 检查用户名是否已存在
	_, err := s.userRepo.GetByUsername(username)
	if err == nil {
		return errors.New("用户名已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查邮箱是否已存在（邮箱非空时）
	if email != "" {
		_, err = s.userRepo.GetByEmail(email)
		if err == nil {
			return errors.New("邮箱已被注册")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询邮箱失败: %w", err)
		}
	}

	// 加密密码
	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	user := &model.User{
		Username: username,
		Password: hashedPassword,
		Email:    email,
		Status:   1,
	}
	return s.userRepo.Create(user)
}

// LoginResult 登录成功后的返回数据。
type LoginResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// Login 用户登录：验证用户名密码 → 检查状态 → 生成 JWT token。
func (s *AuthService) Login(username, password string) (*LoginResult, error) {
	// 查询用户
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 验证密码
	if !crypto.ComparePassword(user.Password, password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 检查用户状态
	if user.Status == 0 {
		return nil, errors.New("账号已被禁用")
	}

	// 生成 JWT token
	token, err := jwtPkg.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}

	return &LoginResult{
		Token: token,
		User:  user,
	}, nil
}

// Logout 用户登出：将 token 加入 Redis 黑名单，使其立即失效。
func (s *AuthService) Logout(token string) error {
	// 解析 token 获取过期时间，计算剩余有效期
	claims, err := jwtPkg.ParseToken(token)
	if err != nil {
		return nil // token 已无效，无需加入黑名单
	}

	// 计算 token 剩余有效期，作为 Redis key 的过期时间
	expiration := time.Until(claims.ExpiresAt.Time)
	if expiration <= 0 {
		return nil // token 已过期
	}

	// 将 token 加入 Redis 黑名单
	key := fmt.Sprintf("token:blacklist:%s", token)
	ctx := context.Background()
	return database.GetRedis().Set(ctx, key, "1", expiration).Err()
}

// IsTokenBlacklisted 检查 token 是否在黑名单中。
func (s *AuthService) IsTokenBlacklisted(token string) bool {
	key := fmt.Sprintf("token:blacklist:%s", token)
	ctx := context.Background()
	result, err := database.GetRedis().Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

// GetUserInfo 根据用户 ID 获取用户信息。
func (s *AuthService) GetUserInfo(userID int64) (*model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}

// ChangePassword 修改密码：验证旧密码 → 校验新密码复杂度 → 加密新密码 → 更新数据库。
func (s *AuthService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	// 密码复杂度校验
	if err := checkPasswordComplexity(newPassword); err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if !crypto.ComparePassword(user.Password, oldPassword) {
		return errors.New("原密码错误")
	}

	// 加密新密码
	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	user.Password = hashedPassword
	return s.userRepo.Update(user)
}

// checkPasswordComplexity 校验密码复杂度：
// - 长度 8-50 位
// - 必须包含大写字母、小写字母、数字
func checkPasswordComplexity(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度不能少于8位")
	}
	if len(password) > 50 {
		return errors.New("密码长度不能超过50位")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含数字")
	}
	return nil
}
