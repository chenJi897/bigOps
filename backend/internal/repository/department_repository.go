package repository

import (
	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/database"
)

type DepartmentRepository struct{}

func NewDepartmentRepository() *DepartmentRepository {
	return &DepartmentRepository{}
}

func (r *DepartmentRepository) Create(dept *model.Department) error {
	return database.GetDB().Create(dept).Error
}

func (r *DepartmentRepository) GetByID(id int64) (*model.Department, error) {
	var dept model.Department
	if err := database.GetDB().First(&dept, id).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepository) GetByName(name string) (*model.Department, error) {
	var dept model.Department
	if err := database.GetDB().Where("name = ?", name).First(&dept).Error; err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *DepartmentRepository) Update(dept *model.Department) error {
	return database.GetDB().Save(dept).Error
}

func (r *DepartmentRepository) Delete(id int64) error {
	return database.GetDB().Delete(&model.Department{}, id).Error
}

func (r *DepartmentRepository) List(page, size int) ([]*model.Department, int64, error) {
	var departments []*model.Department
	var total int64
	db := database.GetDB().Model(&model.Department{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Order("sort ASC, id ASC").Find(&departments).Error; err != nil {
		return nil, 0, err
	}
	return departments, total, nil
}

// GetAll 查询所有启用的部门（用于下拉选择）。
func (r *DepartmentRepository) GetAll() ([]*model.Department, error) {
	var departments []*model.Department
	err := database.GetDB().Where("status = 1").Order("sort ASC, id ASC").Find(&departments).Error
	return departments, err
}

// CountUsersByDepartmentID 统计部门下的用户数。
func (r *DepartmentRepository) CountUsersByDepartmentID(deptID int64) (int64, error) {
	var count int64
	err := database.GetDB().Table("users").Where("department_id = ? AND deleted_at IS NULL", deptID).Count(&count).Error
	return count, err
}
