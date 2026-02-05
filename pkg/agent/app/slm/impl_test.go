package slm

import (
	"context"
	"errors"
	"testing"

	"github.com/abrander/go-supervisord"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeClient struct {
	infos      []*supervisord.ProcessInfo
	infoIdx    int
	processErr error
	startErr   error
	stopErr    error
	startCalls int
	stopCalls  int
}

func (f *fakeClient) nextInfo() *supervisord.ProcessInfo {
	if len(f.infos) == 0 {
		return &supervisord.ProcessInfo{}
	}
	if f.infoIdx >= len(f.infos) {
		return f.infos[len(f.infos)-1]
	}
	info := f.infos[f.infoIdx]
	f.infoIdx++
	return info
}

func (f *fakeClient) StartProcess(name string, wait bool) error {
	f.startCalls++
	return f.startErr
}

func (f *fakeClient) StopProcess(name string, wait bool) error {
	f.stopCalls++
	return f.stopErr
}

func (f *fakeClient) GetProcessInfo(name string) (*supervisord.ProcessInfo, error) {
	if f.processErr != nil {
		return nil, f.processErr
	}
	return f.nextInfo(), nil
}

func newServiceWithClient(fc *fakeClient) *service {
	return &service{
		logger: zap.NewNop().Sugar(),
		client: fc,
	}
}

func TestCheckProcessStarted(t *testing.T) {
	svc := newServiceWithClient(&fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateRunning}},
	})
	_, err := svc.CheckProcessStarted(context.Background(), nil)
	require.NoError(t, err)
}

func TestCheckProcessStartedError(t *testing.T) {
	svc := newServiceWithClient(&fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateStopped}},
	})
	_, err := svc.CheckProcessStarted(context.Background(), nil)
	require.Error(t, err)
}

func TestCheckProcessStopped(t *testing.T) {
	svc := newServiceWithClient(&fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateStopped}},
	})
	_, err := svc.CheckProcessStopped(context.Background(), nil)
	require.NoError(t, err)
}

func TestCheckProcessStoppedError(t *testing.T) {
	svc := newServiceWithClient(&fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateRunning}},
	})
	_, err := svc.CheckProcessStopped(context.Background(), nil)
	require.Error(t, err)
}

func TestStartProcessWhenRunning(t *testing.T) {
	fc := &fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateRunning}},
	}
	svc := newServiceWithClient(fc)
	_, err := svc.StartProcess(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 0, fc.startCalls)
}

func TestStartProcessWhenStopped(t *testing.T) {
	fc := &fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateStopped}},
	}
	svc := newServiceWithClient(fc)
	_, err := svc.StartProcess(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 1, fc.startCalls)
}

func TestStopProcessWhenStopped(t *testing.T) {
	fc := &fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateStopped}},
	}
	svc := newServiceWithClient(fc)
	_, err := svc.StopProcess(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 0, fc.stopCalls)
}

func TestStopProcessWhenRunning(t *testing.T) {
	fc := &fakeClient{
		infos: []*supervisord.ProcessInfo{{State: supervisord.StateRunning}},
	}
	svc := newServiceWithClient(fc)
	_, err := svc.StopProcess(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, 1, fc.stopCalls)
}

func TestGetProcessInfoError(t *testing.T) {
	svc := newServiceWithClient(&fakeClient{processErr: errors.New("fail")})
	_, err := svc.getProcessInfo()
	require.Error(t, err)
}
