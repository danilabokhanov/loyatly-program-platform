package mockstorage

import (
	smimpl "authservice/auth_storage/storage_manager"
	userkeys "authservice/auth_storage/user_keys"
	usermodel "authservice/auth_storage/user_model"
	"sync"

	"github.com/google/uuid"
)

type credentialsData struct {
	Password [userkeys.Md5Len]byte
	UserId   uuid.UUID
}

type MockStorage struct {
	data        map[uuid.UUID]usermodel.User
	credentials map[string]credentialsData
	mx          sync.RWMutex
}

func (ms *MockStorage) GetUserPasswordByLogin(login string) ([userkeys.Md5Len]byte, bool, error) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	cd, ok := ms.credentials[login]
	return cd.Password, ok, nil
}

func (ms *MockStorage) GetUserById(userId uuid.UUID) (usermodel.User, error) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	return ms.data[userId], nil
}

func (ms *MockStorage) GetUserByLogin(login string) (usermodel.User, error) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	cd, ok := ms.credentials[login]
	if !ok {
		return usermodel.User{}, nil
	}
	return ms.data[cd.UserId], nil
}

func (ms *MockStorage) GetNewUUID() uuid.UUID {
	userId := uuid.New()
	for ms.data[userId].Login != "" {
		userId = uuid.New()
	}
	return userId
}

func (ms *MockStorage) AddUser(user usermodel.User, login string, password [userkeys.Md5Len]byte) (uuid.UUID, error) {
	user.ID = ms.GetNewUUID()
	ms.mx.Lock()
	defer ms.mx.Unlock()
	ms.credentials[login] = credentialsData{Password: password, UserId: user.ID}
	ms.data[user.ID] = user
	return user.ID, nil
}

func (ms *MockStorage) UpdateUser(user usermodel.User) error {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	ms.data[user.ID] = user
	return nil
}

func NewStorage() smimpl.Storage {
	return &MockStorage{
		data:        make(map[uuid.UUID]usermodel.User),
		credentials: make(map[string]credentialsData),
	}
}
