package tests

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	FirstName    string    `json:"first_name"`
	SecondName   string    `json:"second_name"`
	BirthDate    time.Time `json:"birth_date"`
	Email        string    `json:"email"`
	PhoneNumber  string    `json:"phone_number"`
	IsCompany    bool      `json:"is_company"`
	CreationDate time.Time `json:"creation_date"`
	UpdateDate   time.Time `json:"update_date"`
	Login        string    `json:"login"`
}

type UserCreds struct {
	Email     string `json:"email"`
	Login     string `json:"login"`
	Password  string `json:"password"`
	IsCompany bool   `json:"is_company"`
}

type StorageManager interface {
	CreateUser(login, password, email string, isCompany bool) (User, error)
	GetJWTByCredentials(login, password string) (string, error)
	GetUserByJWT(jwt string) (User, error)
	UpdateUserByJWT(jwt string, userInfo User) (User, error)
	GetUserById(userId uuid.UUID) (User, error)
}

func IsValidLogin(login string) bool {
	return len(login) >= 6 && strings.IndexFunc(login, func(r rune) bool {
		return r > 127
	}) == -1
}

func IsValidPassword(password string) bool {
	return len(password) >= 8 && strings.IndexFunc(password, unicode.IsDigit) != -1 &&
		strings.IndexFunc(password, unicode.IsUpper) != -1
}

func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(email)
}

func IsValidPhoneNumber(phoneNumber string) bool {
	return phoneNumber[0] == '8' && strings.IndexFunc(phoneNumber, func(r rune) bool {
		return !unicode.IsDigit(r)
	}) == -1
}

func NewUser(login, email, password string, isCompany bool) *User {
	if !IsValidLogin(login) || !IsValidPassword(password) {
		return nil
	}
	curTime := time.Now()
	return &User{
		Login:        login,
		Email:        email,
		IsCompany:    isCompany,
		CreationDate: curTime,
		UpdateDate:   curTime,
	}
}

func MergeUserInfo(user User, newInfo User) User {
	curTime := time.Now()
	if newInfo.FirstName != "" {
		user.FirstName = newInfo.FirstName
	}
	if newInfo.SecondName != "" {
		user.SecondName = newInfo.SecondName
	}
	if !newInfo.BirthDate.IsZero() {
		user.BirthDate = newInfo.BirthDate
	}
	if newInfo.Email != "" && IsValidEmail(newInfo.Email) {
		user.Email = newInfo.Email
	}
	if newInfo.PhoneNumber != "" && IsValidPhoneNumber(newInfo.PhoneNumber) {
		user.PhoneNumber = newInfo.PhoneNumber
	}
	user.UpdateDate = curTime
	return user
}

func FetchUserPublicInfo(user User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		IsCompany: user.IsCompany,
		Login:     user.Login,
	}
}
