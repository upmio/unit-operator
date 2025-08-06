package template

import (
	"github.com/kelseyhightower/memkv"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends"
	"os"
	"time"
)

// Mock implementations for StoreClient and memkv.Store
var MockStoreClient backends.StoreClient

//func TestNewTemplateResource(t *testing.T) {
//	tests := []struct {
//		name      string
//		config    Config
//		wantErr   bool
//		errString string
//	}{
//		//{
//		//	name: "Valid StoreClient",
//		//	config: Config{
//		//		StoreClient:  MockStoreClient,
//		//		TemplateFile: "template.yaml",
//		//		DestFile:     "dest.yaml",
//		//	},
//		//	wantErr: false,
//		//},
//		{
//			name: "Nil StoreClient",
//			config: Config{
//				StoreClient:  nil,
//				TemplateFile: "template.yaml",
//				DestFile:     "dest.yaml",
//			},
//			wantErr:   true,
//			errString: "A valid StoreClient is required.",
//		},
//		//{
//		//	name: "Empty TemplateFile",
//		//	config: Config{
//		//		StoreClient:  MockStoreClient,
//		//		TemplateFile: "",
//		//		DestFile:     "dest.yaml",
//		//	},
//		//	wantErr: false,
//		//},
//		{
//			name: "Empty DestFile",
//			config: Config{
//				StoreClient:  MockStoreClient,
//				TemplateFile: "template.yaml",
//				DestFile:     "",
//			},
//			wantErr: false,
//		},
//		{
//			name: "Valid with Custom Uid and Gid",
//			config: Config{
//				StoreClient:  MockStoreClient,
//				TemplateFile: "template.yaml",
//				DestFile:     "dest.yaml",
//			},
//			wantErr: false,
//		},
//		{
//			name: "Valid with FileMode",
//			config: Config{
//				StoreClient:  MockStoreClient,
//				TemplateFile: "template.yaml",
//				DestFile:     "dest.yaml",
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tr, err := NewTemplateResource(tt.config)
//			if tt.wantErr {
//				assert.Error(t, err)
//				assert.Equal(t, tt.errString, err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, tr)
//				assert.Equal(t, tt.config.TemplateFile, tr.Src)
//				assert.Equal(t, tt.config.DestFile, tr.Dest)
//				assert.Equal(t, uid, tr.Uid)
//				assert.Equal(t, gid, tr.Gid)
//				assert.Equal(t, fileMode, tr.Mode)
//				assert.Equal(t, tt.config.StoreClient, tr.storeClient)
//				assert.NotNil(t, tr.funcMap)
//				//assert.NotNil(t, tr.store)
//			}
//		})
//	}
//}

// Mock utility functions
var resourceTestUtil = struct {
	IsFileExist func(string) bool
}{
	IsFileExist: func(filename string) bool {
		return false
	},
}

//func TestSetFileMode(t *testing.T) {
//	tests := []struct {
//		name      string
//		template  *TemplateResource
//		setup     func()
//		wantMode  os.FileMode
//		wantErr   bool
//		errString string
//	}{
//		{
//			name: "File does not exist, valid mode",
//			template: &TemplateResource{
//				Dest: "nonexistent_file",
//				Mode: "0644",
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//			},
//			wantMode: 0644,
//			wantErr:  false,
//		},
//		{
//			name: "File does not exist, invalid mode",
//			template: &TemplateResource{
//				Dest: "nonexistent_file",
//				Mode: "invalid",
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//			},
//			wantErr:   true,
//			errString: `strconv.ParseUint: parsing "invalid": invalid syntax`,
//		},
//		{
//			name: "File exists, valid mode",
//			template: &TemplateResource{
//				Dest: "existent_file",
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return true
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return &mockFileInfo{mode: 0755}, nil
//				//}
//			},
//			wantMode: 0755,
//			wantErr:  false,
//		},
//		{
//			name: "File exists, stat error",
//			template: &TemplateResource{
//				Dest: "existent_file",
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return true
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return nil, os.ErrNotExist
//				//}
//			},
//			wantErr:   true,
//			errString: "file does not exist",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setup()
//			err := tt.template.setFileMode()
//			if tt.wantErr {
//				assert.Error(t, err)
//				assert.Equal(t, tt.errString, err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.Equal(t, tt.wantMode, tt.template.FileMode)
//			}
//		})
//	}
//}

// Mock implementation of os.FileInfo for testing purposes
type mockFileInfo struct {
	mode os.FileMode
}

func (m *mockFileInfo) Name() string       { return "mockFile" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// Mock implementations for StoreClient and memkv.Store
type TestSetVarsMockStoreClient struct {
	values map[string]string
	err    error
}

func (m *TestSetVarsMockStoreClient) GetValues() (map[string]string, error) {
	return m.values, m.err
}

//type MockStore struct {
//	data map[string]string
//}

var MockStore memkv.Store

//func (m *MockStore) Purge() {
//	m.data = make(map[string]string)
//}
//
//func (m *MockStore) Set(key, value string) {
//	m.data[key] = value
//}

//func TestSetVars(t *testing.T) {
//	tests := []struct {
//		name      string
//		template  *TemplateResource
//		setup     func(*TemplateResource)
//		wantErr   bool
//		errString string
//		wantData  map[string]string
//	}{
//		{
//			name: "Successful setVars",
//			template: &TemplateResource{
//				storeClient: &TestSetVarsMockStoreClient{
//					values: map[string]string{
//						"key1": "value1",
//						"key2": "value2",
//					},
//					err: nil,
//				},
//				//store: MockStore,
//			},
//			setup:   func(tr *TemplateResource) {},
//			wantErr: false,
//			wantData: map[string]string{
//				"key1": "value1",
//				"key2": "value2",
//			},
//		},
//		{
//			name: "GetValues error",
//			template: &TemplateResource{
//				storeClient: &TestSetVarsMockStoreClient{
//					values: nil,
//					err:    errors.New("get values error"),
//				},
//				//store: &MockStore{},
//			},
//			setup:     func(tr *TemplateResource) {},
//			wantErr:   true,
//			errString: "get values error",
//			wantData:  map[string]string{},
//		},
//		{
//			name: "Empty values map",
//			template: &TemplateResource{
//				storeClient: &TestSetVarsMockStoreClient{
//					values: map[string]string{},
//					err:    nil,
//				},
//				//store: &MockStore{},
//			},
//			setup:    func(tr *TemplateResource) {},
//			wantErr:  false,
//			wantData: map[string]string{},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setup(tt.template)
//			err := tt.template.setVars()
//			if tt.wantErr {
//				assert.Error(t, err)
//				assert.Equal(t, tt.errString, err.Error())
//			} else {
//				assert.NoError(t, err)
//				//assert.Equal(t, tt.wantData, tt.template.store.(*MockStore).data)
//			}
//		})
//	}
//}

type TestCreateStageFileStoreClient struct {
	values map[string]string
	err    error
}

func (m *TestCreateStageFileStoreClient) GetValues() (map[string]string, error) {
	return m.values, m.err
}

//func TestCreateStageFile(t *testing.T) {
//	tests := []struct {
//		name      string
//		template  *TemplateResource
//		setup     func()
//		verify    func(*testing.T, *TemplateResource)
//		wantErr   bool
//		errString string
//	}{
//		{
//			name: "Successful createStageFile",
//			template: &TemplateResource{
//				Src:      "template.tmpl",
//				Dest:     "dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{
//						"key1": "value1",
//						"key2": "value2",
//					},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	if strings.Contains(name, "template.tmpl") {
//				//		return &mockFileInfo{mode: 0644}, nil
//				//	}
//				//	return nil, os.ErrNotExist
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return os.Create(pattern)
//				//}
//				//os.Mkdir = func(name string, perm os.FileMode) error {
//				//	return nil
//				//}
//				//os.Chown = func(name string, uid, gid int) error {
//				//	return nil
//				//}
//				//os.Chmod = func(name string, mode os.FileMode) error {
//				//	return nil
//				//}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.NotNil(t, tr.StageFile)
//				assert.FileExists(t, tr.StageFile.Name())
//			},
//			wantErr: false,
//		},
//		{
//			name: "Template parse error",
//			template: &TemplateResource{
//				Src:      "invalid_template.tmpl",
//				Dest:     "dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	if strings.Contains(name, "invalid_template.tmpl") {
//				//		return nil, fmt.Errorf("invalid template")
//				//	}
//				//	return nil, os.ErrNotExist
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return os.Create(pattern)
//				//}
//			},
//			wantErr:   true,
//			errString: "Unable to process template invalid_template.tmpl, invalid template",
//		},
//		{
//			name: "Directory creation error",
//			template: &TemplateResource{
//				Src:      "template.tmpl",
//				Dest:     "nonexistent_dir/dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return nil, os.ErrNotExist
//				//}
//				//os.Mkdir = func(name string, perm os.FileMode) error {
//				//	if strings.Contains(name, "nonexistent_dir") {
//				//		return fmt.Errorf("directory creation error")
//				//	}
//				//	return nil
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return os.Create(pattern)
//				//}
//			},
//			wantErr:   true,
//			errString: "Create nonexistent_dir directory fail, error: directory creation error",
//		},
//		{
//			name: "Chown error",
//			template: &TemplateResource{
//				Src:      "template.tmpl",
//				Dest:     "nonexistent_dir/dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return nil, os.ErrNotExist
//				//}
//				//os.Mkdir = func(name string, perm os.FileMode) error {
//				//	return nil
//				//}
//				//os.Chown = func(name string, uid, gid int) error {
//				//	if strings.Contains(name, "nonexistent_dir") {
//				//		return fmt.Errorf("chown error")
//				//	}
//				//	return nil
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return os.Create(pattern)
//				//}
//			},
//			wantErr:   true,
//			errString: "Chown nonexistent_dir directory fail, error: chown error",
//		},
//		{
//			name: "Temp file creation error",
//			template: &TemplateResource{
//				Src:      "template.tmpl",
//				Dest:     "dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return nil, os.ErrNotExist
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return nil, fmt.Errorf("temp file creation error")
//				//}
//			},
//			wantErr:   true,
//			errString: "temp file creation error",
//		},
//		{
//			name: "Template execution error",
//			template: &TemplateResource{
//				Src:      "template.tmpl",
//				Dest:     "dest.txt",
//				FileMode: 0644,
//				Uid:      1000,
//				Gid:      1000,
//				funcMap:  template.FuncMap{},
//				//store:    MockStore,
//				storeClient: &TestCreateStageFileStoreClient{
//					values: map[string]string{},
//				},
//			},
//			setup: func() {
//				resourceTestUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Stat = func(name string) (os.FileInfo, error) {
//				//	return nil, os.ErrNotExist
//				//}
//				//os.CreateTemp = func(dir, pattern string) (*os.File, error) {
//				//	return os.Create(pattern)
//				//}
//				//template.New = func(name string) *template.Template {
//				//	tmpl := template.Template{}
//				//	tmpl.Execute = func(wr io.Writer, data interface{}) error {
//				//		return fmt.Errorf("template execution error")
//				//	}
//				//	return &tmpl
//				//}
//			},
//			wantErr:   true,
//			errString: "template execution error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setup()
//			err := tt.template.createStageFile()
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

// Mock implementations for utility functions and structs
var testSyncUtil = struct {
	IsConfigChanged func(string, string) (bool, error)
	IsFileExist     func(string) bool
}{
	IsConfigChanged: func(staged, dest string) (bool, error) {
		return true, nil
	},
	IsFileExist: func(filename string) bool {
		return false
	},
}

//func TestSync(t *testing.T) {
//	tests := []struct {
//		name      string
//		template  *TemplateResource
//		setup     func()
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				testSyncUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Rename = func(oldpath, newpath string) error {
//				//	return nil
//				//}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.FileExists(t, tr.Dest)
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				testSyncUtil.IsFileExist = func(filename string) bool {
//					if filename == "dest.txt" {
//						return true
//					}
//					return false
//				}
//				//os.Rename = func(oldpath, newpath string) error {
//				//	return nil
//				//}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.FileExists(t, tr.Dest)
//				assert.FileExists(t, ".previous_dest.txt")
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return false, fmt.Errorf("isConfigChanged error")
//				}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				testSyncUtil.IsFileExist = func(filename string) bool {
//					return true
//				}
//				//os.Rename = func(oldpath, newpath string) error {
//				//	if newpath == ".previous_dest.txt" {
//				//		return fmt.Errorf("rename error")
//				//	}
//				//	return nil
//				//}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				testSyncUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Rename = func(oldpath, newpath string) error {
//				//	return fmt.Errorf("device or resource busy")
//				//}
//				//ioutil.ReadFile = func(filename string) ([]byte, error) {
//				//	return []byte("content"), nil
//				//}
//				//ioutil.WriteFile = func(filename string, data []byte, perm os.FileMode) error {
//				//	return nil
//				//}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.FileExists(t, tr.Dest)
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
//			setup: func() {
//				testSyncUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				testSyncUtil.IsFileExist = func(filename string) bool {
//					return false
//				}
//				//os.Rename = func(oldpath, newpath string) error {
//				//	return fmt.Errorf("device or resource busy")
//				//}
//				//ioutil.ReadFile = func(filename string) ([]byte, error) {
//				//	return []byte("content"), nil
//				//}
//				//ioutil.WriteFile = func(filename string, data []byte, perm os.FileMode) error {
//				//	return fmt.Errorf("write file error")
//				//}
//				//os.Remove = func(name string) error {
//				//	return nil
//				//}
//			},
//			wantErr:   true,
//			errString: "write file error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setup()
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

//func TestProcess(t *testing.T) {
//	tests := []struct {
//		name      string
//		config    Config
//		setup     func(*MockFileSystem, *MockStoreClient)
//		verify    func(*testing.T, *TemplateResource)
//		wantErr   bool
//		errString string
//	}{
//		{
//			name: "Successful process",
//			config: Config{
//				StoreClient:  &MockStoreClient{values: map[string]interface{}{"key": "value"}},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			setup: func(mfs *MockFileSystem, msc *MockStoreClient) {
//				mfs.files["template.txt"] = []byte("template content")
//				resourceTestUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//			},
//			verify: func(t *testing.T, tr *TemplateResource) {
//				assert.True(t, tr.fs.IsExist(tr.Dest))
//			},
//			wantErr: false,
//		},
//		{
//			name: "Error in setFileMode",
//			config: Config{
//				StoreClient:  &MockStoreClient{},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			setup: func(mfs *MockFileSystem, msc *MockStoreClient) {
//				mfs.existing["dest.txt"] = true
//				mfs.renameError[".previous_dest.txt"] = errors.New("rename error")
//			},
//			wantErr:   true,
//			errString: "rename error",
//		},
//		{
//			name: "Error in setVars",
//			config: Config{
//				StoreClient:  &MockStoreClient{err: errors.New("get values error")},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			wantErr:   true,
//			errString: "get values error",
//		},
//		{
//			name: "Error in createStageFile",
//			config: Config{
//				StoreClient:  &MockStoreClient{values: map[string]interface{}{"key": "value"}},
//				TemplateFile: "invalid_template.txt",
//				DestFile:     "dest.txt",
//			},
//			wantErr:   true,
//			errString: "Unable to process template invalid_template.txt",
//		},
//		{
//			name: "Error in sync",
//			config: Config{
//				StoreClient:  &MockStoreClient{values: map[string]interface{}{"key": "value"}},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			setup: func(mfs *MockFileSystem, msc *MockStoreClient) {
//				mfs.files["template.txt"] = []byte("template content")
//				resourceTestUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
//					return true, nil
//				}
//				mfs.renameError["dest.txt"] = errors.New("rename error")
//			},
//			wantErr:   true,
//			errString: "rename error",
//		},
//		{
//			name: "Fallback to WriteFile after device or resource busy error",
//			config: Config{
//				StoreClient:  &MockStoreClient{values: map[string]interface{}{"key": "value"}},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			setup: func(mfs *MockFileSystem, msc *MockStoreClient) {
//				mfs.files["template.txt"] = []byte("template content")
//				resourceTestUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
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
//			config: Config{
//				StoreClient:  &MockStoreClient{values: map[string]interface{}{"key": "value"}},
//				TemplateFile: "template.txt",
//				DestFile:     "dest.txt",
//			},
//			setup: func(mfs *MockFileSystem, msc *MockStoreClient) {
//				mfs.files["template.txt"] = []byte("template content")
//				resourceTestUtil.IsConfigChanged = func(staged, dest string) (bool, error) {
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
//			msc := &MockStoreClient{}
//			if tt.setup != nil {
//				tt.setup(mfs, msc)
//			}
//
//			tr, err := NewTemplateResource(tt.config, mfs)
//			assert.NoError(t, err)
//
//			err = tr.process()
//			if tt.wantErr {
//				assert.Error(t, err)
//				assert.Equal(t, tt.errString, err.Error())
//			} else {
//				assert.NoError(t, err)
//				if tt.verify != nil {
//					tt.verify(t, tr)
//				}
//			}
//		})
//	}
//}
