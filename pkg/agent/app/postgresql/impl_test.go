package postgresql

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeStorageFactory struct {
	data []byte
	err  error
}

func (f *fakeStorageFactory) PutFile(context.Context, string, string, string) error {
	return nil
}

func (f *fakeStorageFactory) GetFile(context.Context, string, string, string) error {
	return nil
}

func (f *fakeStorageFactory) PutObject(context.Context, string, string, io.Reader) error {
	return nil
}

func (f *fakeStorageFactory) GetObject(context.Context, string, string) (io.ReadCloser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(bytes.NewReader(f.data)), nil
}

func newPostgresService(t *testing.T) *service {
	t.Helper()
	return &service{
		logger: zap.NewNop().Sugar(),
	}
}

func buildTarArchive(t *testing.T, entries map[string]string) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range entries {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(header))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	return buf.Bytes()
}

func TestSafeTarPath(t *testing.T) {
	svc := newPostgresService(t)
	path, err := svc.safeTarPath("/tmp", "data/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join("/tmp", "data/file.txt"), path)

	_, err = svc.safeTarPath("/tmp", "../etc/passwd")
	require.Error(t, err)
}

func TestExtractTarEntryCreatesFileAndDir(t *testing.T) {
	svc := newPostgresService(t)
	dir := t.TempDir()

	// directory entry
	dirTar := bytes.NewBuffer(nil)
	tw := tar.NewWriter(dirTar)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "config", Typeflag: tar.TypeDir, Mode: 0o755}))
	require.NoError(t, tw.Close())

	tr := tar.NewReader(bytes.NewReader(dirTar.Bytes()))
	hdr, err := tr.Next()
	require.NoError(t, err)
	require.NoError(t, svc.extractTarEntry(tr, hdr, dir))
	require.DirExists(t, filepath.Join(dir, "config"))

	// file entry
	var fileBuf bytes.Buffer
	tw = tar.NewWriter(&fileBuf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "config/postgresql.conf", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len("test"))}))
	_, err = tw.Write([]byte("test"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	tr = tar.NewReader(bytes.NewReader(fileBuf.Bytes()))
	hdr, err = tr.Next()
	require.NoError(t, err)
	require.NoError(t, svc.extractTarEntry(tr, hdr, dir))
	data, err := os.ReadFile(filepath.Join(dir, "config/postgresql.conf"))
	require.NoError(t, err)
	require.Equal(t, "test", string(data))
}

func TestExtractFileFromS3(t *testing.T) {
	svc := newPostgresService(t)
	dir := t.TempDir()

	tarData := buildTarArchive(t, map[string]string{
		"base/file.txt": "content",
	})

	fakeFactory := &fakeStorageFactory{data: tarData}

	err := svc.extractFileFromS3(context.Background(), fakeFactory, "bucket", "backup", "base.tar", dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, "base/file.txt"))
	require.NoError(t, err)
	require.Equal(t, "content", string(content))
}

func TestRemoveContents(t *testing.T) {
	svc := newPostgresService(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("data"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested/b.txt"), []byte("data"), 0o644))

	require.NoError(t, svc.removeContents(dir))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 0)
}
