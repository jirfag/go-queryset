package gorm2

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

func queryUsersWithMaxRating(db *gorm.DB, limit int) *gorm.DB {
	return db.Order("rating DESC").Limit(limit)
}

func queryUsersRegisteredToday(db *gorm.DB, limit int) *gorm.DB {
	today := getTodayBegin()
	return db.Where("created_at >= ?", today).Limit(limit)
}

// GetUsersWithMaxRating returns limit users with maximal rating
func GetUsersWithMaxRating(limit int) ([]User, error) {
	var users []User
	if err := queryUsersWithMaxRating(getGormDB(), limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersRegisteredToday returns all users registered today
func GetUsersRegisteredToday(limit int) ([]User, error) {
	var users []User
	if err := queryUsersRegisteredToday(getGormDB(), limit).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersRegisteredTodayWithMaxRating returns all users
// registered today with max rating
func GetUsersRegisteredTodayWithMaxRating(limit int) ([]User, error) {
	var users []User
	err := queryUsersWithMaxRating(queryUsersRegisteredToday(getGormDB(), limit), limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
