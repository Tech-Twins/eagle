package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateID generates a unique ID with the given prefix
func GenerateID(prefix string) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 10

	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return fmt.Sprintf("%s-%s", prefix, string(result))
}

// GenerateAccountNumber generates an 8-digit account number starting with 01
func GenerateAccountNumber() string {
	num, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("01%06d", num.Int64())
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if a password matches a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateAccountNumber validates the account number format
func ValidateAccountNumber(accountNumber string) bool {
	return len(accountNumber) == 8 && strings.HasPrefix(accountNumber, "01")
}

// ValidateUserID validates the user ID format
func ValidateUserID(userID string) bool {
	return strings.HasPrefix(userID, "usr-")
}

// ValidateTransactionID validates the transaction ID format
func ValidateTransactionID(transactionID string) bool {
	return strings.HasPrefix(transactionID, "tan-")
}
