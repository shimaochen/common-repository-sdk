package repository

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Filter 筛选结构体
type Filter struct {
	Filterable []string               //可供筛选的字段
	QueryStr   string                 //接口url传的query字符串
	Filters    map[string]interface{} //业务逻辑中使用
	Sortable   []string               //可供排序的字段
	Sort       string
	Page       int
	PageSize   int
	Unscoped   bool         //是否包含软删除的记录
	Joins      []JoinConfig //支持 JOIN
	sqlRecords []string
	Debug      bool
	finalSQL   string
}

// JoinConfig JOIN 配置结构
type JoinConfig struct {
	Table    string // 要 join 的表，例如 "roles"
	On       string // 连接条件，例如 "users.role_id = roles.id"
	JoinType string // "left" 或 "inner"
}

// PaginationQuery 主入口
func (f *Filter) PaginationQuery(db *gorm.DB) *gorm.DB {
	if f.Debug {
		f.sqlRecords = []string{}
	}

	// 先处理 Unscoped（软删除）
	if f.Unscoped {
		db = db.Unscoped()
		f.recordSQL("UNSCOPED", "include soft-deleted records")
	}

	// 执行 JOIN
	if len(f.Joins) > 0 {
		for _, j := range f.Joins {
			switch strings.ToLower(j.JoinType) {
			case "left":
				db = db.Joins(fmt.Sprintf("LEFT JOIN %s ON %s", j.Table, j.On))
				f.recordSQL(fmt.Sprintf("LEFT JOIN %s ON %s", j.Table, j.On), nil)
			default:
				db = db.Joins(fmt.Sprintf("INNER JOIN %s ON %s", j.Table, j.On))
				f.recordSQL(fmt.Sprintf("INNER JOIN %s ON %s", j.Table, j.On), nil)
			}
		}
	}

	// Filters条件
	if len(f.Filters) > 0 {
		db = f.applyQueryConditions(db, f.Filters)
	}
	// 动态条件
	if f.QueryStr != "" {
		var queryMap map[string]interface{}
		if err := json.Unmarshal([]byte(f.QueryStr), &queryMap); err == nil {
			db = f.applyQueryConditions(db, queryMap)
		}
	}

	return db
}

// ================== 内部函数 ==================

// 应用查询条件
func (f *Filter) applyQueryConditions(db *gorm.DB, conditions map[string]interface{}) *gorm.DB {
	for field, value := range conditions {
		// 允许 "表名.字段名"
		if !f.isFilterable(field) {
			continue
		}
		switch v := value.(type) {
		case string, int, float64, bool:
			db = db.Where(fmt.Sprintf("%s = ?", field), v)
			f.recordSQL(fmt.Sprintf("EQ %s", field), v)
		case []interface{}:
			db = db.Where(fmt.Sprintf("%s IN (?)", field), v)
			f.recordSQL(fmt.Sprintf("IN %s", field), v)
		case []string:
			db = db.Where(fmt.Sprintf("%s IN (?)", field), v)
			f.recordSQL(fmt.Sprintf("IN %s", field), v)
		case map[string]interface{}:
			db = f.applyComplexCondition(db, field, v)
		}
	}
	return db
}

// 应用复杂条件（如 like、gt、between）
func (f *Filter) applyComplexCondition(db *gorm.DB, field string, conds map[string]interface{}) *gorm.DB {
	for op, value := range conds {
		switch op {
		case "eq":
			db = db.Where(fmt.Sprintf("%s = ?", field), value)
			f.recordSQL(fmt.Sprintf("EQ %s", field), value)
		case "neq":
			db = db.Where(fmt.Sprintf("%s != ?", field), value)
			f.recordSQL(fmt.Sprintf("NEQ %s", field), value)
		case "gt":
			db = db.Where(fmt.Sprintf("%s > ?", field), value)
			f.recordSQL(fmt.Sprintf("GT %s", field), value)
		case "gte":
			db = db.Where(fmt.Sprintf("%s >= ?", field), value)
			f.recordSQL(fmt.Sprintf("GTE %s", field), value)
		case "lt":
			db = db.Where(fmt.Sprintf("%s < ?", field), value)
			f.recordSQL(fmt.Sprintf("LT %s", field), value)
		case "lte":
			db = db.Where(fmt.Sprintf("%s <= ?", field), value)
			f.recordSQL(fmt.Sprintf("LTE %s", field), value)
		case "like":
			db = db.Where(fmt.Sprintf("%s LIKE ?", field), fmt.Sprintf("%v", value))
			f.recordSQL(fmt.Sprintf("LIKE %s", field), value)
		case "in":
			db = db.Where(fmt.Sprintf("%s IN (?)", field), value)
			f.recordSQL(fmt.Sprintf("IN %s", field), value)
		case "between":
			if arr, ok := value.([]interface{}); ok && len(arr) == 2 {
				db = db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), arr[0], arr[1])
				f.recordSQL(fmt.Sprintf("BETWEEN %s", field), arr)
			}
		}
	}
	return db
}

// ApplySortAndPagination 排序分页
func (f *Filter) ApplySortAndPagination(db *gorm.DB) *gorm.DB {
	// 排序
	if f.Sort != "" {
		for _, s := range strings.Split(f.Sort, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			order := "ASC"
			field := s
			if strings.HasPrefix(s, "-") {
				order = "DESC"
				field = strings.TrimPrefix(s, "-")
			}
			if f.isSortable(field) {
				db = db.Order(fmt.Sprintf("%s %s", field, order))
				f.recordSQL(fmt.Sprintf("ORDER %s %s", field, order), nil)
			}
		}
	}

	// 分页
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 10
	}
	if f.PageSize > 500 {
		f.PageSize = 500
	}
	offset := (f.Page - 1) * f.PageSize
	db = db.Offset(offset).Limit(f.PageSize)
	f.recordSQL("Pagination", map[string]int{"page": f.Page, "pageSize": f.PageSize})
	if f.Debug {
		sql := db.Session(&gorm.Session{DryRun: true}).ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Find(nil)
		})
		f.finalSQL = sql
	}
	return db
}

// 记录调试 SQL
func (f *Filter) recordSQL(desc string, val interface{}) {
	if !f.Debug {
		return
	}
	f.sqlRecords = append(f.sqlRecords, fmt.Sprintf("[%s] | args: %v", desc, val))
}

// PrintSQLs 打印调试信息
func (f *Filter) PrintSQLs() {
	fmt.Println("=== Generated SQL Statements ===")
	for i, s := range f.sqlRecords {
		fmt.Printf("%d. %s\n", i+1, s)
	}
	if f.finalSQL != "" {
		fmt.Println("---------------------------------")
		fmt.Printf("[Final SQL Preview]\n%s\n", f.finalSQL)
	}
	fmt.Println("=================================")
}

func (f *Filter) isFilterable(field string) bool {
	if len(f.Filterable) == 0 {
		return true
	}
	for _, w := range f.Filterable {
		if w == field {
			return true
		}
	}
	return false
}

func (f *Filter) isSortable(field string) bool {
	if strings.TrimSpace(field) == "" {
		return false
	}
	// 特判：id 字段总是可排序的
	if field == "id" || field == "created_at" || field == "updated_at" {
		return true
	}

	if len(f.Sortable) == 0 {
		return false
	}

	for _, w := range f.Sortable {
		if w == field {
			return true
		}
	}
	return false
}
