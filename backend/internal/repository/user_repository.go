// Package repository 提供数据访问层，封装对数据库的 CRUD 操作。
package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

// UserRepository 用户数据访问对象。
type UserRepository struct{}

// NewUserRepository 创建 UserRepository 实例。
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create 创建新用户。
func (r *UserRepository) Create(user *model.User) error {
	return database.GetDB().Create(user).Error
}

// GetByID 根据用户 ID 查询用户。
func (r *UserRepository) GetByID(id int64) (*model.User, error) {
	var user model.User
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名查询用户。
func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	if err := database.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱查询用户。
func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	if err := database.GetDB().Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户信息（仅更新非零值字段）。
func (r *UserRepository) Update(user *model.User) error {
	return database.GetDB().Save(user).Error
}

// Delete 软删除用户。
func (r *UserRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.User{}, id).Error
}

// List 分页查询用户列表，返回用户列表和总数。
func (r *UserRepository) List(page, size int) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	db := database.GetDB().Model(&model.User{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
