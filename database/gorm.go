package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func NewGormDB(dialecter gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialecter, &gorm.Config{
		DisableAutomaticPing: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
}
