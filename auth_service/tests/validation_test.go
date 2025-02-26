package tests

import (
	usermodel "authservice/auth_storage/user_model"
	"testing"
)

func TestIsValidLogin(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"user2024", true},
		{"usr", false},
		{"логин", false},
		{"validUser_99", true},
	}

	for _, test := range tests {
		if result := usermodel.IsValidLogin(test.input); result != test.expected {
			t.Errorf("IsValidLogin(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Secr3tP@ss", true},
		{"12345678", false},
		{"n0uppercase", false},
		{"lowercase1", false},
		{"G00dP@ssw0rd", true},
	}

	for _, test := range tests {
		if result := usermodel.IsValidPassword(test.input); result != test.expected {
			t.Errorf("IsValidPassword(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"contact@domain.com", true},
		{"missingat.com", false},
		{"@nodomain.com", false},
		{"name.surname@company.org", true},
	}

	for _, test := range tests {
		if result := usermodel.IsValidEmail(test.input); result != test.expected {
			t.Errorf("IsValidEmail(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestIsValidPhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"89001234567", true},
		{"70001234567", false},
		{"8phone12345", false},
		{"81112223344", true},
	}

	for _, test := range tests {
		if result := usermodel.IsValidPhoneNumber(test.input); result != test.expected {
			t.Errorf("IsValidPhoneNumber(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
