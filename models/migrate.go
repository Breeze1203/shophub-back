package models

import "gorm.io/gorm"

func AutoMigrateAll(db *gorm.DB) error {
	err := db.AutoMigrate(
		&User{},
		&Room{},
		&Message{},
		&CustomerSession{},
	)
	if err != nil {
		return err
	}
	return nil
}
