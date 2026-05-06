package xcasbin

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// create a new mysql database instance
func newDatabase(dsn string, logger ...Logger) (*gorm.DB, error) {
	var opts []gorm.Option
	if len(logger) > 0 && logger[0] != nil {
		opts = []gorm.Option{&gorm.Config{Logger: logger[0]}}
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), opts...)
	if err != nil {
		return nil, err
	}

	if d, err := db.DB(); err != nil {
		return nil, err
	} else {
		d.SetMaxIdleConns(20)
		d.SetMaxOpenConns(20)
		d.SetConnMaxLifetime(2 * time.Minute)
	}

	return db, nil
}
