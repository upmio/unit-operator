package patch

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGenerateMergePatch(t *testing.T) {
	old := &TestStruct{
		Name: "Old Name",
		Age:  20,
	}

	update := &TestStruct{
		Name: "New Name",
		Age:  21,
	}

	dataStruct := &TestStruct{}

	patchBytes, changed, err := GenerateMergePatch(old, update, dataStruct)

	assert.NoError(t, err)
	assert.True(t, changed)
	assert.NotEmpty(t, patchBytes)
}

func TestGenerateMergePatch_NoChange(t *testing.T) {
	old := &TestStruct{
		Name: "Old Name",
		Age:  20,
	}

	update := &TestStruct{
		Name: "Old Name",
		Age:  20,
	}

	dataStruct := &TestStruct{}

	_, changed, err := GenerateMergePatch(old, update, dataStruct)

	assert.NoError(t, err)
	assert.False(t, changed)
	//assert.Empty(t, patchBytes)
}
