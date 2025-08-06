package proxysql

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TestMarshalToGSetVariableRequest(t *testing.T) {
	data := map[string]interface{}{
		"key":      "auto_increment",
		"value":    "2",
		"section":  "admin",
		"username": "root",
		"password": "password",
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req SetVariableRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "auto_increment", req.GetKey(), "Key mismatch")
	require.Equal(t, "2", req.GetValue(), "Value mismatch")
	require.Equal(t, "admin", req.GetSection(), "SocketFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
}
