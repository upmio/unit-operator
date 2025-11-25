package common

import (
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"os"
	"testing"
)

func TestGetPlainTextPassword(t *testing.T) {
	os.Setenv(vars.AESEnvKey, "7097029b4c29f2cf6c796361fc174d77")

	if err := util.ValidateAndSetAESKey(); err != nil {
		t.Fatal(err)
	}

	plaintext, err := GetPlainTextPassword("gXx37xofWmHc41KwbxZ6CO1TlSOSlJDS2hciy+cVHxU=")
	if err != nil {
		t.Error(err)
	}

	if plaintext != "sJPk1@028@&5&@lA" {
		t.Errorf("plain text is wrong")
	}
}
