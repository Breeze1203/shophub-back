package models

import "gorm.io/gorm"

func AutoMigrateAll(db *gorm.DB) error {
	err := db.AutoMigrate(
		&User{},
		&Room{},
		&Message{},
		&CustomerSession{},
		&PetCategory{},
		&Pet{},
		&PetImage{},
		&PetSpecification{},
		&Discount{},
		&PetDiscount{},
		&Coupon{},
		&CategoryCoupon{},
		&PetCoupon{},
		&UserCoupon{},
	)
	if err != nil {
		return err
	}
	return nil
}
