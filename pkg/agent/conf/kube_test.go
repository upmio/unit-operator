package conf

import (
	"testing"

	"github.com/stretchr/testify/require"
	unitv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

func TestBuildUnitClientSchemeRegistersUnitV1alpha2(t *testing.T) {
	scheme, err := buildUnitClientScheme()
	require.NoError(t, err)

	_, err = scheme.New(unitv1alpha2.GroupVersion.WithKind("Unit"))
	require.NoError(t, err)
}
