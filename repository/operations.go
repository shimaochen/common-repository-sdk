package repository

import (
	"errors"

	"gorm.io/gorm"
)

// GetInfoById 通用的根据id获取详细
func GetInfoById[T any](db *gorm.DB, id uint) (*T, error) {
	if id == 0 {
		return nil, errors.New("id cannot be zero")
	}
	var res *T
	err := db.Model(new(T)).
		Where("id = ?", id).
		Last(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Created 创建
func Created[T any](db *gorm.DB, m *T) error {
	return db.Create(m).Error
}

// UpdateByIdWithMap 通用的根据ID删除记录
func UpdateByIdWithMap[T any](db *gorm.DB, id uint, updates map[string]interface{}) error {
	if id == 0 {
		return errors.New("id cannot be zero")
	}

	result := db.Model(new(T)).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// QueryWithPagination 通用分页查询函数
func QueryWithPagination[T any](db *gorm.DB, f *Filter) ([]T, int64, int, int, error) {
	var (
		result []T
		count  int64
	)
	queryDB := f.PaginationQuery(db.Model(new(T)))
	if err := queryDB.Count(&count).Error; err != nil {
		return nil, 0, f.Page, f.PageSize, err
	}
	if count == 0 {
		return []T{}, 0, f.Page, f.PageSize, nil
	}
	queryDB = f.ApplySortAndPagination(queryDB)
	if f.Debug {
		f.PrintSQLs()
	}
	if err := queryDB.Find(&result).Error; err != nil {
		return nil, 0, f.Page, f.PageSize, err
	}

	return result, count, f.Page, f.PageSize, nil
}

// QueryWithFilter 通用查询函数
func QueryWithFilter[T any](db *gorm.DB, f *Filter) ([]T, error) {
	var result []T
	queryDB := f.PaginationQuery(db.Model(new(T)))
	queryDB = f.ApplySortAndPagination(queryDB)
	// SQL日志
	if f.Debug {
		f.PrintSQLs()
	}

	if err := queryDB.Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// SoftDeleteById 通用的根据ID删除记录,   DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
func SoftDeleteById[T any](db *gorm.DB, id uint) error {
	if id == 0 {
		return errors.New("id cannot be zero")
	}

	result := db.Delete(new(T), id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// DeleteById 设置is_deleted = 1
func DeleteById[T any](db *gorm.DB, id uint) error {
	if id == 0 {
		return errors.New("id cannot be zero")
	}

	result := db.Model(new(T)).
		Where("id = ?", id).
		UpdateColumn("is_deleted", 1)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func GetDB[T any](db *gorm.DB) *gorm.DB {
	return db.Model(new(T))
}
