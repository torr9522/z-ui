package password

import "golang.org/x/crypto/bcrypt"

func Hash(plain string) (string, error) {
	data, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Verify(hash string, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
