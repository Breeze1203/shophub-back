package models

import "gorm.io/gorm"

func AutoMigrateAll(db *gorm.DB) error {
	err := db.AutoMigrate(
		&User{},
		&Room{},
		&Message{},
	)
	if err != nil {
		return err
	}
	return nil
}
