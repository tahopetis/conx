package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("invalid password hash")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong     = errors.New("password must be no more than 128 characters")
	ErrPasswordMissingUpper = errors.New("password must contain at least one uppercase letter")
	ErrPasswordMissingLower = errors.New("password must contain at least one lowercase letter")
	ErrPasswordMissingNumber = errors.New("password must contain at least one number")
	ErrPasswordMissingSpecial = errors.New("password must contain at least one special character")
)

type PasswordConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type PasswordService struct {
	config PasswordConfig
}

func NewPasswordService(config PasswordConfig) *PasswordService {
	return &PasswordService{
		config: config,
	}
}

func DefaultPasswordConfig() PasswordConfig {
	return PasswordConfig{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func (s *PasswordService) HashPassword(password string) (string, error) {
	// Validate password before hashing
	if err := s.ValidatePasswordStrength(password); err != nil {
		return "", err
	}

	// Generate a random salt
	salt := make([]byte, s.config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.config.Iterations,
		s.config.Memory,
		s.config.Parallelism,
		s.config.KeyLength,
	)

	// Base64 encode the salt and hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return the formatted hash
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.config.Memory,
		s.config.Iterations,
		s.config.Parallelism,
		b64Salt,
		b64Hash,
	), nil
}

func (s *PasswordService) VerifyPassword(password, hash string) (bool, error) {
	// Parse the hash
	parsedHash, err := s.parseHash(hash)
	if err != nil {
		return false, err
	}

	// Hash the provided password with the same parameters
	otherHash := argon2.IDKey(
		[]byte(password),
		parsedHash.salt,
		parsedHash.iterations,
		parsedHash.memory,
		parsedHash.parallelism,
		parsedHash.keyLength,
	)

	// Compare the hashes in constant time to prevent timing attacks
	if subtle.ConstantTimeCompare(parsedHash.hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

func (s *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	if len(password) > 128 {
		return ErrPasswordTooLong
	}

	if !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ErrPasswordMissingUpper
	}

	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		return ErrPasswordMissingLower
	}

	if !strings.ContainsAny(password, "0123456789") {
		return ErrPasswordMissingNumber
	}

	if !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return ErrPasswordMissingSpecial
	}

	return nil
}

func (s *PasswordService) GeneratePassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	const (
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		numberChars  = "0123456789"
		specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars     = upperChars + lowerChars + numberChars + specialChars
	)

	password := make([]byte, length)
	
	// Ensure at least one character from each category
	password[0] = upperChars[rand.Intn(len(upperChars))]
	password[1] = lowerChars[rand.Intn(len(lowerChars))]
	password[2] = numberChars[rand.Intn(len(numberChars))]
	password[3] = specialChars[rand.Intn(len(specialChars))]
	
	// Fill the rest with random characters from all categories
	for i := 4; i < length; i++ {
		password[i] = allChars[rand.Intn(len(allChars))]
	}
	
	// Shuffle the password
	for i := range password {
		j := rand.Intn(len(password))
		password[i], password[j] = password[j], password[i]
	}
	
	return string(password), nil
}

type parsedHash struct {
	version     uint32
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt        []byte
	hash        []byte
	keyLength   uint32
}

func (s *PasswordService) parseHash(encodedHash string) (*parsedHash, error) {
	// Example hash: $argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, ErrInvalidHash
	}

	if parts[0] != "$argon2id" {
		return nil, ErrIncompatibleVersion
	}

	var version uint32
	_, err := fmt.Sscanf(parts[1], "v=%d", &version)
	if err != nil {
		return nil, ErrInvalidHash
	}

	if version != argon2.Version {
		return nil, ErrIncompatibleVersion
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err = fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, ErrInvalidHash
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, ErrInvalidHash
	}

	return &parsedHash{
		version:     version,
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		salt:        salt,
		hash:        hash,
		keyLength:   uint32(len(hash)),
	}, nil
}

// CheckPasswordCommonPatterns checks if the password is too common or follows predictable patterns
func (s *PasswordService) CheckPasswordCommonPatterns(password string) error {
	// Convert to lowercase for pattern matching
	lowerPassword := strings.ToLower(password)

	// Check for common passwords
	commonPasswords := []string{
		"password", "123456", "12345678", "123456789", "12345",
		"qwerty", "abc123", "letmein", "admin", "welcome",
		"password123", "admin123", "qwerty123", "123qwe",
	}

	for _, common := range commonPasswords {
		if lowerPassword == common {
			return errors.New("password is too common")
		}
	}

	// Check for sequential patterns
	if s.isSequential(lowerPassword) {
		return errors.New("password contains sequential characters")
	}

	// Check for repeated characters
	if s.isRepeated(lowerPassword) {
		return errors.New("password contains too many repeated characters")
	}

	// Check for keyboard patterns
	if s.isKeyboardPattern(lowerPassword) {
		return errors.New("password follows a keyboard pattern")
	}

	return nil
}

func (s *PasswordService) isSequential(password string) bool {
	// Check for ascending/descending sequences
	sequences := []string{
		"abcdefghijklmnopqrstuvwxyz",
		"zyxwvutsrqponmlkjihgfedcba",
		"0123456789",
		"987654321",
	}

	for _, seq := range sequences {
		if strings.Contains(password, seq) {
			return true
		}
	}

	return false
}

func (s *PasswordService) isRepeated(password string) bool {
	// Check if the same character is repeated more than 3 times
	repeated := regexp.MustCompile(`(.)\1{3,}`)
	return repeated.MatchString(password)
}

func (s *PasswordService) isKeyboardPattern(password string) bool {
	// Common keyboard patterns
	patterns := []string{
		"qwerty", "asdfgh", "zxcvbn",
		"qazwsx", "edcrfv", "tgbnhy",
		"1qaz", "2wsx", "3edc", "4rfv", "5tgb", "6yhn", "7ujm",
	}

	lowerPassword := strings.ToLower(password)
	for _, pattern := range patterns {
		if strings.Contains(lowerPassword, pattern) {
			return true
		}
	}

	return false
}

// GetPasswordStrength returns a strength score (0-100) and feedback for the password
func (s *PasswordService) GetPasswordStrength(password string) (int, string) {
	score := 0
	feedback := []string{}

	// Length
	if len(password) >= 8 {
		score += 20
	} else {
		feedback = append(feedback, "Use at least 8 characters")
	}

	if len(password) >= 12 {
		score += 10
	}

	if len(password) >= 16 {
		score += 10
	}

	// Character variety
	if strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		score += 10
	} else {
		feedback = append(feedback, "Add uppercase letters")
	}

	if strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") {
		score += 10
	} else {
		feedback = append(feedback, "Add lowercase letters")
	}

	if strings.ContainsAny(password, "0123456789") {
		score += 10
	} else {
		feedback = append(feedback, "Add numbers")
	}

	if strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		score += 10
	} else {
		feedback = append(feedback, "Add special characters")
	}

	// Deduct points for common patterns
	if err := s.CheckPasswordCommonPatterns(password); err != nil {
		score -= 20
		feedback = append(feedback, "Avoid common patterns")
	}

	// Cap the score at 100
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	// Generate feedback message
	var feedbackMsg string
	if len(feedback) == 0 {
		if score >= 80 {
			feedbackMsg = "Strong password"
		} else if score >= 60 {
			feedbackMsg = "Good password"
		} else {
			feedbackMsg = "Fair password"
		}
	} else {
		feedbackMsg = "Suggestions: " + strings.Join(feedback, ", ")
	}

	return score, feedbackMsg
}
