package smimpl

import (
	userkeys "authservice/auth_storage/user_keys"
	usermodel "authservice/auth_storage/user_model"

	"github.com/google/uuid"
)

type Storage interface {
	GetUserPasswordByLogin(login string) ([userkeys.Md5Len]byte, bool, error)
	GetUserById(userId uuid.UUID) (usermodel.User, error)
	GetUserByLogin(login string) (usermodel.User, error)
	AddUser(user usermodel.User, login string, password [userkeys.Md5Len]byte) (uuid.UUID, error)
	UpdateUser(user usermodel.User) error
}

type StorageManager struct {
	storage Storage
}

func (sm *StorageManager) CreateUser(login, password, email string, isCompany bool) (usermodel.User, error) {
	if user, err := sm.storage.GetUserByLogin(login); err != nil || user.Login != "" {
		return usermodel.User{}, err
	}
	user := usermodel.NewUser(login, email, password, isCompany)
	if user == nil {
		return usermodel.User{}, nil
	}
	hashedPassword := userkeys.GetPasswordHash(login, password)
	userId, err := sm.storage.AddUser(*user, login, hashedPassword)
	if err != nil {
		return usermodel.User{}, err
	}
	user.ID = userId
	return *user, nil
}

func (sm *StorageManager) GetJWTByCredentials(login, password string) (string, error) {
	expectedPassword, ok, err := sm.storage.GetUserPasswordByLogin(login)
	if err != nil || !ok {
		return "", err
	}
	actualPassword := userkeys.GetPasswordHash(login, password)
	if actualPassword != expectedPassword {
		return "", nil
	}
	user, err := sm.storage.GetUserByLogin(login)
	if err != nil {
		return "", err
	}
	return userkeys.GenJWT(user.ID), nil
}

func (sm *StorageManager) GetUserByJWT(jwt string) (usermodel.User, error) {
	userId, ok := userkeys.GetUserIdByJWT(jwt)
	if !ok {
		return usermodel.User{}, nil
	}
	return sm.storage.GetUserById(userId)
}

func (sm *StorageManager) UpdateUserByJWT(jwt string, userInfo usermodel.User) (usermodel.User, error) {
	user, err := sm.GetUserByJWT(jwt)
	if err != nil || user.Login == "" {
		return usermodel.User{}, err
	}
	user = usermodel.MergeUserInfo(user, userInfo)
	return user, sm.storage.UpdateUser(user)
}

func (sm *StorageManager) GetUserById(userId uuid.UUID) (usermodel.User, error) {
	user, err := sm.storage.GetUserById(userId)
	if err != nil {
		return usermodel.User{}, err
	}
	return usermodel.FetchUserPublicInfo(user), nil
}

func NewStorageManager(storage Storage) usermodel.StorageManager {
	return &StorageManager{
		storage: storage,
	}
}
