package mongodb

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
	stoppedErr error
}

func (f *fakeSlm) CheckProcessStarted(context.Context, *common.Empty) (*common.Empty, error) {
	return nil, f.startedErr
}

func (f *fakeSlm) CheckProcessStopped(context.Context, *common.Empty) (*common.Empty, error) {
	return nil, f.stoppedErr
}

func newMongoServiceWithSlm(startErr, stopErr error) *service {
	return &service{
		logger: zap.NewNop().Sugar(),
		slm:    &fakeSlm{startedErr: startErr, stoppedErr: stopErr},
	}
}

func TestBackupFailsWhenProcessNotStarted(t *testing.T) {
	startErr := errors.New("not running")
	svc := newMongoServiceWithSlm(startErr, nil)

	_, err := svc.Backup(context.Background(), &BackupRequest{
		Username:      "user",
		ObjectStorage: &common.ObjectStorage{},
	})

	require.Equal(t, startErr, err)
}

func TestRestoreFailsWhenProcessNotStarted(t *testing.T) {
	startErr := errors.New("not running")
	svc := newMongoServiceWithSlm(startErr, nil)

	_, err := svc.Restore(context.Background(), &RestoreRequest{
		Username:      "user",
		ObjectStorage: &common.ObjectStorage{},
	})

	require.Equal(t, startErr, err)
}
