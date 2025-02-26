package pgstorage

import (
	smimpl "authservice/auth_storage/storage_manager"
	userkeys "authservice/auth_storage/user_keys"
	usermodel "authservice/auth_storage/user_model"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const PGCredentialsPath = "credentials/postgresql.json"

type UserInfo struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	FirstName    string    `gorm:"type:varchar(255)"`
	SecondName   string    `gorm:"type:varchar(255)"`
	BirthDate    time.Time `gorm:"type:date;not null"`
	Email        string    `gorm:"type:varchar(255);unique;not null"`
	PhoneNumber  string    `gorm:"type:varchar(20)"`
	IsCompany    bool      `gorm:"not null"`
	CreationDate time.Time `gorm:"autoCreateTime"`
	UpdateDate   time.Time `gorm:"autoUpdateTime"`
}

type UserCredentials struct {
	Login    string    `gorm:"type:varchar(255);primaryKey"`
	Password []byte    `gorm:"type:bytea;not null"`
	UserID   uuid.UUID `gorm:"type:uuid;not null"`
	User     UserInfo  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type UserWithLogin struct {
	UserInfo
	Login string `gorm:"type:varchar(255)"`
}

type PGCredentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"db_name"`
}

var (
	pgCreds PGCredentials
	once    sync.Once
)

type PGStorage struct {
	db *gorm.DB
}

func GetUserInfoByUser(user usermodel.User) *UserInfo {
	return &UserInfo{
		ID:           user.ID,
		FirstName:    user.FirstName,
		SecondName:   user.SecondName,
		BirthDate:    user.BirthDate,
		Email:        user.Email,
		PhoneNumber:  user.PhoneNumber,
		IsCompany:    user.IsCompany,
		CreationDate: user.CreationDate,
		UpdateDate:   user.UpdateDate,
	}
}

func (ps *PGStorage) GetUserPasswordByLogin(login string) ([userkeys.Md5Len]byte, bool, error) {
	var userCreds UserCredentials
	err := ps.db.First(&userCreds, "login = ?", login).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return [userkeys.Md5Len]byte{}, false, nil
	}
	return [userkeys.Md5Len]byte(userCreds.Password), userCreds.Login != "", err
}

func (ps *PGStorage) GetUserById(userId uuid.UUID) (usermodel.User, error) {
	var user usermodel.User
	err := ps.db.
		Table("user_info").
		Select("user_info.id, user_info.first_name, user_info.second_name, user_info.birth_date, user_info.email, user_info.phone_number, user_info.is_company, user_info.creation_date, user_info.update_date, user_credentials.login").
		Joins("JOIN user_credentials ON user_info.id = user_credentials.user_id").
		Where("user_info.id = ?", userId).
		Scan(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return usermodel.User{}, nil
	}
	if err != nil {
		return usermodel.User{}, err
	}
	return user, nil
}

func (ps *PGStorage) GetUserByLogin(login string) (usermodel.User, error) {
	var user usermodel.User
	err := ps.db.
		Table("user_info").
		Select("user_info.id, user_info.first_name, user_info.second_name, user_info.birth_date, user_info.email, user_info.phone_number, user_info.is_company, user_info.creation_date, user_info.update_date, user_credentials.login").
		Joins("JOIN user_credentials ON user_info.id = user_credentials.user_id").
		Where("user_credentials.login = ?", login).
		Scan(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return usermodel.User{}, nil
	}
	return user, err
}

func (ps *PGStorage) AddUser(user usermodel.User, login string, password [userkeys.Md5Len]byte) (uuid.UUID, error) {
	userInfo := GetUserInfoByUser(user)
	err := ps.db.Create(&userInfo).Error
	if err != nil {
		return uuid.UUID{}, err
	}
	userCreds := UserCredentials{Login: login, Password: password[:], UserID: userInfo.ID}
	err = ps.db.Create(&userCreds).Error
	if err != nil {
		return uuid.UUID{}, err
	}
	return userInfo.ID, nil
}

func (ps *PGStorage) UpdateUser(user usermodel.User) error {
	userInfo := GetUserInfoByUser(user)
	err := ps.db.Save(&userInfo).Error
	return err
}

func getPostgresCreds() *PGCredentials {
	once.Do(func() {
		file, err := os.Open(PGCredentialsPath)
		if err != nil {
			log.Fatalf("Error opening PostgreSQL credentials file: %v", err)
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(&pgCreds)
		if err != nil {
			if err != nil {
				log.Fatalf("Error parsing PostgreSQL credentials: %v", err)
			}
		}
	})
	return &pgCreds
}

func NewStorage() smimpl.Storage {
	pgCreds := getPostgresCreds()
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		"postgres", 5432, pgCreds.User, pgCreds.Password, pgCreds.DBName, "disable")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		log.Fatalf("Error running PostgreSQL: %v", err)
	}
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	err = db.AutoMigrate(&UserInfo{}, &UserCredentials{})
	if err != nil {
		log.Fatalf("Failed setting up schema: %v", err)
	}
	fmt.Println("Established successful connection to PostgreSQL")
	return &PGStorage{db: db}
}
