package project

import "testing"

func TestGenerateAES256Key(t *testing.T) {
	key, err := generateAES256Key()
	if err != nil {
		t.Fatalf("generateAES256Key fail: %v", err)
	}

	if len(key) != 32 {

		t.Errorf("the generated key length is not 32, actual: %d", len(key))
	}
	// Check if it is a valid 32-byte hexadecimal string
	if !isValidHex32(key) {
		t.Errorf("the generated key is not a valid 32-byte hexadecimal string: %s", string(key))
	}
}
