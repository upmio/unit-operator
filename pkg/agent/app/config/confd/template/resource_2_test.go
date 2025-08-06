package template

import (
	"io/ioutil"
	"os"
	"time"
)

// FileSystem defines an interface for file system operations
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	Remove(name string) error
	Rename(oldpath, newpath string) error
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	Chmod(name string, mode os.FileMode) error
	Chown(name string, uid, gid int) error
	IsExist(name string) bool
}

// OSFileSystem implements FileSystem using the os package
type OSFileSystem struct{}

func (OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (OSFileSystem) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func (OSFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

func (OSFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (OSFileSystem) Chown(name string, uid, gid int) error {
	return os.Chown(name, uid, gid)
}

func (OSFileSystem) IsExist(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

//// TemplateResource represents the template resource
//type TemplateResource struct {
//	Dest        string
//	FileMode    os.FileMode
//	Gid         int
//	Mode        string
//	Src         string
//	StageFile   *os.File
//	Uid         int
//	funcMap     map[string]interface{}
//	store       memkv.Store
//	storeClient backends.StoreClient
//	fs          FileSystem
//}
//
//// NewTemplateResource creates a new TemplateResource
//func NewTemplateResource(config Config, fs FileSystem) (*TemplateResource, error) {
//	if config.StoreClient == nil {
//		return nil, errors.New("A valid StoreClient is required.")
//	}
//
//	tr := &TemplateResource{
//		Src:         config.TemplateFile,
//		Dest:        config.DestFile,
//		Uid:         uid,
//		Gid:         gid,
//		Mode:        fileMode,
//		storeClient: config.StoreClient,
//		funcMap:     newFuncMap(),
//		store:       memkv.New(),
//		fs:          fs,
//	}
//
//	addFuncs(tr.funcMap, tr.store.FuncMap)
//
//	return tr, nil
//}
//
//// sync synchronizes the template resource
//func (t *TemplateResource) sync() error {
//	staged := t.StageFile.Name()
//
//	defer t.fs.Remove(staged)
//	ok, err := resourceTestUtil.IsConfigChanged(staged, t.Dest)
//	if err != nil {
//		return err
//	}
//
//	if ok {
//		if t.fs.IsExist(t.Dest) {
//			previousFileName := fmt.Sprintf(".previous_%s", filepath.Base(t.Dest))
//			previousFile := filepath.Join(filepath.Dir(t.Dest), previousFileName)
//			err := t.fs.Rename(t.Dest, previousFile)
//			if err != nil {
//				return err
//			}
//		}
//
//		err = t.fs.Rename(staged, t.Dest)
//		if err != nil {
//			if strings.Contains(err.Error(), "device or resource busy") {
//				// try to open the file and write to it
//				var contents []byte
//				var rerr error
//				contents, rerr = t.fs.ReadFile(staged)
//				if rerr != nil {
//					return rerr
//				}
//				err := t.fs.WriteFile(t.Dest, contents, t.FileMode)
//				// make sure owner and group match the temp file, in case the file was created with WriteFile
//				t.fs.Chown(t.Dest, t.Uid, t.Gid)
//				if err != nil {
//					return err
//				}
//			} else {
//				return err
//			}
//		}
//
//	}
//	return nil
//}

// MockFileSystem is a mock implementation of FileSystem
type MockFileSystem struct {
	files       map[string][]byte
	existing    map[string]bool
	renameError map[string]error
	readError   map[string]error
	writeError  map[string]error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string][]byte),
		existing:    make(map[string]bool),
		renameError: make(map[string]error),
		readError:   make(map[string]error),
		writeError:  make(map[string]error),
	}
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.existing[name] {
		return &testSyncMockFileInfo{name: name}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) Remove(name string) error {
	delete(m.existing, name)
	delete(m.files, name)
	return nil
}

func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	if err, ok := m.renameError[newpath]; ok {
		return err
	}
	m.files[newpath] = m.files[oldpath]
	delete(m.files, oldpath)
	m.existing[newpath] = true
	delete(m.existing, oldpath)
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if err, ok := m.readError[filename]; ok {
		return nil, err
	}
	return m.files[filename], nil
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err, ok := m.writeError[filename]; ok {
		return err
	}
	m.files[filename] = data
	m.existing[filename] = true
	return nil
}

func (m *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	return nil
}

func (m *MockFileSystem) Chown(name string, uid, gid int) error {
	return nil
}

func (m *MockFileSystem) IsExist(name string) bool {
	return m.existing[name]
}

// mockFileInfo is a mock implementation of os.FileInfo
type testSyncMockFileInfo struct {
	name string
}

func (m *testSyncMockFileInfo) Name() string       { return m.name }
func (m *testSyncMockFileInfo) Size() int64        { return 0 }
func (m *testSyncMockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *testSyncMockFileInfo) ModTime() time.Time { return time.Now() }
func (m *testSyncMockFileInfo) IsDir() bool        { return false }
func (m *testSyncMockFileInfo) Sys() interface{}   { return nil }

//func TestSync2(t *testing.T) {
//	tests := []struct {
//		name      string
//		template  *TemplateResource
//		setup     func(mfs *MockFileSystem)
//		verify    func(*testing.T, *TemplateResource)
//		wantErr   bool
//		errString string
//	}{
//		{
//			name: "Successful sync with no previous file",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				mfs.existing["staged.txt"] = true
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.True(t, tr.fs.IsExist(tr.Dest))
//			},
//			wantErr: false,
//		},
//		{
//			name: "Successful sync with previous file",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				mfs.existing["staged.txt"] = true
//				mfs.existing["dest.txt"] = true
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.True(t, tr.fs.IsExist(tr.Dest))
//				assert.True(t, tr.fs.IsExist(".previous_dest.txt"))
//			},
//			wantErr: false,
//		},
//		{
//			name: "Error during IsConfigChanged",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return false, fmt.Errorf("isConfigChanged error")
//				}
//			},
//			wantErr:   true,
//			errString: "isConfigChanged error",
//		},
//		{
//			name: "Error during first os.Rename",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				mfs.existing["staged.txt"] = true
//				mfs.existing["dest.txt"] = true
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				mfs.renameError[".previous_dest.txt"] = errors.New("rename error")
//			},
//			wantErr:   true,
//			errString: "rename error",
//		},
//		{
//			name: "Error during second os.Rename with fallback to WriteFile",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				mfs.existing["staged.txt"] = true
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				mfs.renameError["dest.txt"] = errors.New("device or resource busy")
//				mfs.files["staged.txt"] = []byte("content")
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.True(t, tr.fs.IsExist(tr.Dest))
//			},
//			wantErr: false,
//		},
//		{
//			name: "Error during WriteFile after device or resource busy error",
//			template: &TemplateResource{
//				Dest:      "dest.txt",
//				FileMode:  0644,
//				Uid:       1000,
//				Gid:       1000,
//				StageFile: &os.File{},
//			},
//			setup: func(mfs *MockFileSystem) {
//				mfs.existing["staged.txt"] = true
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				mfs.renameError["dest.txt"] = errors.New("device or resource busy")
//				mfs.readError["staged.txt"] = errors.New("read file error")
//			},
//			wantErr:   true,
//			errString: "read file error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			mfs := NewMockFileSystem()
//			tt.setup(mfs)
//			tt.template.fs = mfs
//
//			err := tt.template.sync()
//			if tt.wantErr {
//				assert.Error(t, err)
//				assert.Equal(t, tt.errString, err.Error())
//			} else {
//				assert.NoError(t, err)
//				if tt.verify != nil {
//					tt.verify(t, tt.template)
//				}
//			}
//		})
//	}
//}
