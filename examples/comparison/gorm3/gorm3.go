package gorm3

import (
	"time"

	"github.com/jinzhu/gorm"
)

func getGormDB() *gorm.DB {
	db, _ := gorm.Open("mysql",
		"user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
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

func queryUsersWithMaxRating(db *gorm.DB) *gorm.DB {
	return db.Order("rating DESC")
}

func queryUsersRegisteredToday(db *gorm.DB) *gorm.DB {
	return db.Where("created_at >= ?", getTodayBegin())
}

// GetUsersWithMaxRating returns limit users with maximal rating
func GetUsersWithMaxRating(limit int) ([]User, error) {
	var users []User
	err := queryUsersWithMaxRating(getGormDB()).
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersRegisteredToday returns all users registered today
func GetUsersRegisteredToday(limit int) ([]User, error) {
	var users []User
	err := queryUsersRegisteredToday(getGormDB()).
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersRegisteredTodayWithMaxRating returns all users
// registered today with max rating
func GetUsersRegisteredTodayWithMaxRating(limit int) ([]User, error) {
	var users []User
	err := getGormDB().
		Scopes(queryUsersWithMaxRating, queryUsersRegisteredToday).
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
