package tests

import (
	usermodel "authservice/auth_storage/user_model"
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		login     string
		email     string
		password  string
		isCompany bool
		expected  bool
	}{
		{"validUser", "test@example.com", "ValidPass123", false, true},
		{"usr", "test@example.com", "ValidPass123", false, false},
		{"validUser", "test@example.com", "short", false, false},
	}

	for _, test := range tests {
		user := usermodel.NewUser(test.login, test.email, test.password, test.isCompany)
		if (user != nil) != test.expected {
			t.Errorf("NewUser(%q, %q, %q, %v) = %v; want %v", test.login, test.email, test.password, test.isCompany, user != nil, test.expected)
		}
	}
}

func TestMergeUserInfo(t *testing.T) {
	oldUser := usermodel.User{
		ID:          uuid.New(),
		FirstName:   "Alice",
		SecondName:  "Brown",
		Email:       "alice.old@example.com",
		PhoneNumber: "89991234567",
	}

	newUser := usermodel.User{
		FirstName:  "Bob",
		SecondName: "Green",
		Email:      "bob.new@example.com",
	}

	updatedUser := usermodel.MergeUserInfo(oldUser, newUser)

	if updatedUser.FirstName != "Bob" || updatedUser.SecondName != "Green" || updatedUser.Email != "bob.new@example.com" {
		t.Errorf("MergeUserInfo did not correctly update fields")
	}
}

func TestFetchUserPublicInfo(t *testing.T) {
	user := usermodel.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		IsCompany: true,
		Login:     "testUser",
	}

	publicUser := usermodel.FetchUserPublicInfo(user)

	if publicUser.ID != user.ID || publicUser.Email != user.Email || publicUser.IsCompany != user.IsCompany || publicUser.Login != user.Login {
		t.Errorf("FetchUserPublicInfo did not return correct public data")
	}
}
