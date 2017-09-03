package base

import (
	"github.com/jinzhu/gorm"
)

var gormDB *gorm.DB

// SetGormDB sets global gormDB instance
func SetGormDB(DB *gorm.DB) {
	gormDB = DB
}

// Base is a base query set struct
type Base struct {
	methods []GormMethod
}

// GormMethod is a method for one query set operation
type GormMethod func(d *gorm.DB) *gorm.DB

// AddGormMethod adds where condition
func (b *Base) AddGormMethod(m GormMethod) {
	b.methods = append(b.methods, m)
}

// GetQuerySet returns gorm query set with set filters
func (b *Base) GetQuerySet() *gorm.DB {
	DB := gormDB
	for _, m := range b.methods {
		DB = m(DB)
	}

	return DB.Unscoped()
}
