package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	password, err := HashPassword("xhxh123123")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(password)
	// $2a$10$8o5M5JQ0ZZJNC8hDKyLwfOOi2cmmKoFKSalOCLMlarAe/3eEscMHK
}

func TestPassword(t *testing.T) {
	password1 := "$2b$10$NGIVWhes789F4LsZX9H7s.8DE4bHbMucgL.TNFPi6tW8rB3IpFnTG"
	password2 := "12345678"
	err := bcrypt.CompareHashAndPassword([]byte(password1), []byte(password2))
	if err != nil {
		t.Fatal(err)
	}
}
