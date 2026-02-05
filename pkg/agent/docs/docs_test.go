package docs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwaggerInfoDefaults(t *testing.T) {
	require.Equal(t, "unit-agent api demo", SwaggerInfo.Title)
	require.Equal(t, "1.0", SwaggerInfo.Version)
	require.Equal(t, "swagger", SwaggerInfo.InstanceName())
}

func TestDocTemplateIncludesSwaggerVersion(t *testing.T) {
	require.True(t, strings.Contains(docTemplate, `"swagger": "2.0"`))
}
