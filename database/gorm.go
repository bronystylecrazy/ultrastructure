package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"go.uber.org/zap"
)

func NewGormDB(log *zap.Logger, dialecter gorm.Dialector) (*gorm.DB, error) {
	log.Debug("opening database")
	return gorm.Open(dialecter, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
}

func NewPostgresDialector(config Config) gorm.Dialector {
	return postgres.Open(config.Datasource)
}
