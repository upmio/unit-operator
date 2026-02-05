package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubStoreClient struct {
	values map[string]string
	err    error
}

func (s *stubStoreClient) GetValues() (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.values, nil
}

func TestNewTemplateResourceRequiresStore(t *testing.T) {
	_, err := NewTemplateResource(Config{})
	require.Error(t, err)
}

func TestSetFileModeDefaults(t *testing.T) {
	tr := &TemplateResource{
		Dest: "/tmp/nonexistent",
		Mode: "0640",
	}
	require.NoError(t, tr.setFileMode())
	require.Equal(t, os.FileMode(0640), tr.FileMode)
}

func TestSetFileModeExistingFile(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.conf")
	require.NoError(t, os.WriteFile(dest, []byte("content"), 0600))

	tr := &TemplateResource{
		Dest: dest,
		Mode: "0644",
	}
	require.NoError(t, tr.setFileMode())
	require.Equal(t, os.FileMode(0600), tr.FileMode)
}

func TestSetVarsPopulatesStore(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "tmpl")
	require.NoError(t, os.WriteFile(templatePath, []byte("value"), 0644))

	cfg := Config{
		StoreClient:  &stubStoreClient{values: map[string]string{"key": "value"}},
		TemplateFile: templatePath,
		DestFile:     filepath.Join(dir, "dest"),
	}

	tr, err := NewTemplateResource(cfg)
	require.NoError(t, err)
	require.NoError(t, tr.setVars())

	kv, err := tr.store.GetValue("key")
	require.NoError(t, err)
	require.Equal(t, "value", kv)
}

func TestCreateStageFile(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "source.tmpl")
	require.NoError(t, os.WriteFile(templatePath, []byte("hello"), 0644))
	destPath := filepath.Join(dir, "conf", "output.conf")

	fakeStore := &stubStoreClient{values: map[string]string{}}
	tr, err := NewTemplateResource(Config{
		StoreClient:  fakeStore,
		TemplateFile: templatePath,
		DestFile:     destPath,
	})
	require.NoError(t, err)

	require.NoError(t, tr.setFileMode())
	require.NoError(t, tr.createStageFile())
	require.NotNil(t, tr.StageFile)
	_, err = os.Stat(tr.StageFile.Name())
	require.NoError(t, err)
}

func TestSyncWritesDestWhenChanged(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.conf")
	require.NoError(t, os.WriteFile(dest, []byte("old"), 0644))

	stageFile, err := os.CreateTemp(dir, "stage")
	require.NoError(t, err)
	_, err = stageFile.Write([]byte("new"))
	require.NoError(t, err)
	require.NoError(t, stageFile.Close())

	tr := &TemplateResource{
		Dest:      dest,
		FileMode:  0644,
		StageFile: stageFile,
	}

	require.NoError(t, tr.sync())

	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "new", string(content))

	_, err = os.Stat(stageFile.Name())
	require.True(t, os.IsNotExist(err))
}
