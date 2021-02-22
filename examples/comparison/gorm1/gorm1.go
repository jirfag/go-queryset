package gorm1

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func getGormDB() *gorm.DB {
	db, _ := gorm.Open(mysql.Open("user:password@/dbname?charset=utf8&parseTime=True&loc=Local"), &gorm.Config{})
	return db
}

// User struct represents user model.
type User struct {
	gorm.Model
	Rating      int
	RatingMarks int
}

func getTodayBegin() time.Time {
	year, month, day := time.Now().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Now().Location())
}

// GetUsersWithMaxRating returns limit users with maximal rating
func GetUsersWithMaxRating(limit int) ([]User, error) {
	var users []User
	if err := getGormDB().Order("rating DESC").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersRegisteredToday returns all users registered today
func GetUsersRegisteredToday(limit int) ([]User, error) {
	var users []User
	today := getTodayBegin()
	err := getGormDB().Where("created_at >= ?", today).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
