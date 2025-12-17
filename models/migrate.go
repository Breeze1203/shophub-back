package models

import "gorm.io/gorm"

func AutoMigrateAll(db *gorm.DB) error {
	err := db.AutoMigrate(
		&User{},
		&Room{},
		&Message{},
		&CustomerSession{},
		&MerchantInfo{},
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
		&Favorite{},
		&Cart{},
		&MerchantFollow{},
	)
	if err != nil {
		return err
	}
	return nil
}
