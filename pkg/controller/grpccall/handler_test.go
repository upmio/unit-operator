package grpccall

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/upmio/unit-operator/pkg/agent/app/mysql"
)

// Test unmarshalParams function
func TestUnmarshalParams(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]apiextensionsv1.JSON
		expectErr bool
	}{
		{
			name: "valid parameters",
			params: map[string]apiextensionsv1.JSON{
				"username": {Raw: []byte(`"root"`)},
				"password": {Raw: []byte(`"password"`)},
			},
			expectErr: false,
		},
		{
			name:      "empty parameters",
			params:    map[string]apiextensionsv1.JSON{},
			expectErr: false,
		},
		{
			name: "invalid JSON",
			params: map[string]apiextensionsv1.JSON{
				"username": {Raw: []byte(`invalid json`)},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &mysql.LogicalBackupRequest{}
			err := unmarshalParams(tt.params, msg)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalParams_ComplexData(t *testing.T) {
	params := map[string]apiextensionsv1.JSON{
		"username": {Raw: []byte(`"root"`)},
		"password": {Raw: []byte(`"mypassword"`)},
		"parallel": {Raw: []byte(`4`)},
	}

	msg := &mysql.PhysicalBackupRequest{}
	err := unmarshalParams(params, msg)

	assert.NoError(t, err)
	assert.Equal(t, "root", msg.GetUsername())
	assert.Equal(t, "mypassword", msg.GetPassword())
	assert.Equal(t, int64(4), msg.GetParallel())
}

func TestUnmarshalParams_InvalidProtobufField(t *testing.T) {
	params := map[string]apiextensionsv1.JSON{
		"nonexistent_field": {Raw: []byte(`"value"`)},
	}

	msg := &mysql.LogicalBackupRequest{}
	err := unmarshalParams(params, msg)

	// protojson.Unmarshal will error on unknown fields
	assert.Error(t, err)
}

func TestUnmarshalParams_InvalidJsonStructure(t *testing.T) {
	// Test with a slice when object is expected
	params := map[string]apiextensionsv1.JSON{
		"username": {Raw: []byte(`["array", "instead", "of", "string"]`)},
	}

	msg := &mysql.LogicalBackupRequest{}
	err := unmarshalParams(params, msg)

	// This should fail because username expects string, not array
	assert.Error(t, err)
}

func TestUnmarshalParams_EmptyRaw(t *testing.T) {
	params := map[string]apiextensionsv1.JSON{
		"username": {Raw: []byte(`""`)}, // empty string is valid
		"password": {Raw: nil},          // nil raw data
	}

	msg := &mysql.LogicalBackupRequest{}
	err := unmarshalParams(params, msg)

	// Should handle nil raw data
	assert.NoError(t, err)
	assert.Equal(t, "", msg.GetUsername())
}

func TestUnmarshalParams_NumericFields(t *testing.T) {
	params := map[string]apiextensionsv1.JSON{
		"parallel": {Raw: []byte(`8`)},
	}

	msg := &mysql.PhysicalBackupRequest{}
	err := unmarshalParams(params, msg)

	assert.NoError(t, err)
	assert.Equal(t, int64(8), msg.GetParallel())
}

func TestUnmarshalParams_BooleanFields(t *testing.T) {
	params := map[string]apiextensionsv1.JSON{
		"compress": {Raw: []byte(`true`)},
		"verbose":  {Raw: []byte(`false`)},
	}

	// Create a generic proto message for testing
	msg := &mysql.LogicalBackupRequest{}
	err := unmarshalParams(params, msg)

	// Should fail because these fields don't exist in the message
	assert.Error(t, err)
}

func TestUnmarshalParams_MarshalError(t *testing.T) {
	// Create a simple test case that should work
	params := map[string]apiextensionsv1.JSON{
		"username": {Raw: []byte(`"valid"`)},
	}

	// Add an invalid field that doesn't exist in protobuf
	invalidParams := map[string]apiextensionsv1.JSON{
		"username":      {Raw: []byte(`"valid"`)},
		"invalid_field": {Raw: []byte(`"invalid"`)},
	}

	// Test the normal case should work
	msg := &mysql.LogicalBackupRequest{}
	err := unmarshalParams(params, msg)
	assert.NoError(t, err)
	assert.Equal(t, "valid", msg.GetUsername())

	// Test invalid field should fail
	msg2 := &mysql.LogicalBackupRequest{}
	err = unmarshalParams(invalidParams, msg2)
	assert.Error(t, err)
}
