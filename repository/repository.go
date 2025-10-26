package repository

import (
	"gorm.io/gorm"
)

type Repository[T any] interface {
	GetInfoById(id uint) (*T, error)
	Create(m *T) error
	UpdateById(id uint, updates map[string]interface{}) error
	DeleteById(id uint) error
	SoftDeleteById(id uint) error
	ListPagination(f *Filter) ([]T, int64, int, int, error)
	ListByFilter(f *Filter) ([]T, error)
	GetDB() *gorm.DB
}

type baseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) Repository[T] {
	return &baseRepository[T]{db: db}
}

func (r *baseRepository[T]) GetInfoById(id uint) (*T, error) {
	return GetInfoById[T](r.db, id)
}

func (r *baseRepository[T]) Create(m *T) error {
	return Created[T](r.db, m)
}

func (r *baseRepository[T]) UpdateById(id uint, updates map[string]interface{}) error {
	return UpdateByIdWithMap[T](r.db, id, updates)
}

func (r *baseRepository[T]) DeleteById(id uint) error {
	return DeleteById[T](r.db, id)
}

func (r *baseRepository[T]) SoftDeleteById(id uint) error {
	return SoftDeleteById[T](r.db, id)
}

func (r *baseRepository[T]) ListPagination(f *Filter) ([]T, int64, int, int, error) {
	return QueryWithPagination[T](r.db, f)
}

func (r *baseRepository[T]) ListByFilter(f *Filter) ([]T, error) {
	return QueryWithFilter[T](r.db, f)
}

func (r *baseRepository[T]) GetDB() *gorm.DB {
	return GetDB[T](r.db)
}
