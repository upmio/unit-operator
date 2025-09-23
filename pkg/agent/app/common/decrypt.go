package common

import (
	"encoding/base64"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
)

func GetPlainTextPassword(password string) (string, error) {
	encrypt, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return "", err
	}

	plaintext, err := util.AES_CTR_Decrypt(encrypt)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
