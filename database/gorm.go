package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func NewGormDB(dialecter gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialecter, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
}

func NewPostgresDialector(config Config) gorm.Dialector {
	return postgres.Open(config.Datasource)
}
