package proxysql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"go.uber.org/zap"
)

type fakeSlm struct {
	slm.UnimplementedServiceLifecycleServer
	startedErr error
}

func (f *fakeSlm) CheckProcessStarted(context.Context, *common.Empty) (*common.Empty, error) {
	return nil, f.startedErr
}

func TestSetVariableFailsWhenProcessDown(t *testing.T) {
	startErr := errors.New("down")
	svc := &service{
		logger: zap.NewNop().Sugar(),
		slm:    &fakeSlm{startedErr: startErr},
	}

	_, err := svc.SetVariable(context.Background(), &SetVariableRequest{
		Username: "user",
		Section:  "admin",
		Key:      "var",
		Value:    "value",
	})

	require.Equal(t, startErr, err)
}

func TestCloseDBConnHandlesNil(t *testing.T) {
	svc := &service{logger: zap.NewNop().Sugar()}
	require.NotPanics(t, func() { svc.closeDBConn(nil) })
}
