package xgorm

import (
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func NewDB(dialecter gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialecter, &gorm.Config{
		DisableAutomaticPing: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
}

func NewSQLDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}
