package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type fakeGrpcApp struct {
	name          string
	configErr     error
	configCalls   int
	registryCalls int
}

func (f *fakeGrpcApp) Registry(*grpc.Server) {
	f.registryCalls++
}

func (f *fakeGrpcApp) Config() error {
	f.configCalls++
	return f.configErr
}

func (f *fakeGrpcApp) Name() string {
	return f.name
}

func resetGrpcApps(t *testing.T) {
	t.Helper()
	grpcApps = map[string]GRPCApp{}
}

func TestRegistryGrpcAppAndLoad(t *testing.T) {
	resetGrpcApps(t)

	app := &fakeGrpcApp{name: "mysql"}
	RegistryGrpcApp(app)

	require.Equal(t, app, GetGrpcApp("mysql"))

	server := grpc.NewServer()
	err := LoadGrpcApp(server)
	require.NoError(t, err)
	require.Equal(t, 1, app.configCalls)
	require.Equal(t, 1, app.registryCalls)

	loaded := LoadedGrpcApp()
	require.Len(t, loaded, 1)
	require.Equal(t, "mysql", loaded[0])
}

func TestRegistryGrpcAppDuplicatePanics(t *testing.T) {
	resetGrpcApps(t)

	app := &fakeGrpcApp{name: "mysql"}
	RegistryGrpcApp(app)

	require.Panics(t, func() {
		RegistryGrpcApp(app)
	})
}

func TestLoadGrpcAppConfigError(t *testing.T) {
	resetGrpcApps(t)

	app := &fakeGrpcApp{name: "mysql", configErr: errors.New("boom")}
	RegistryGrpcApp(app)

	server := grpc.NewServer()
	err := LoadGrpcApp(server)
	require.Error(t, err)
}

func TestGetGrpcAppPanicsWhenMissing(t *testing.T) {
	resetGrpcApps(t)

	require.Panics(t, func() {
		GetGrpcApp("missing")
	})
}

