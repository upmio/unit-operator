package template

import (
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends"
	"io/ioutil"
	"os"
	"testing"
)

const (
	testDir = "./test"
)

const (
	tomlFilePath = "test/confd/config.toml"
	tmplFilePath = "test/templates/test.conf.tmpl"
)

type templateTest struct {
	desc             string                  // description of the test (for helpful errors)
	toml             string                  // toml file contents
	tmpl             string                  // template file contents
	content          string                  // value contents
	extetend_content string                  // value extend contents
	expected         interface{}             // expected generated file contents
	updateStore      func(*TemplateResource) // function for setting values in store
}

// templateTests is an array of templateTest structs, each representing a test of
// some aspect of template processing. When the input tmpl and toml files are
// processed, they should produce a config file matching expected.
var templateTests = []templateTest{
	{
		desc: "base test 1",
		tmpl: `test: {{ getv "/default/test" }}
connect: {{ join (jsonArrayAppend (getv "/default/connect") ":9000" ":2900") "," }}
ip: {{ getenv "HOST_IP" }}
password: {{ AESCBCDecrypt "/ru+KsOJgjj+JZS11HRh1IDFsQILgnyoqn16XqyoKoo=" }}
extend1: {{ getv "/extend1" }}
extend2: {{ getv "/extend2" }}
{{- range $index,$value := jsonArray (getv "/default/save")}}
save {{ $index }} {{ $value }}
{{- end }}
`,
		expected: `test: xxxxxx
connect: aaaa:9000:2900,bbbb:9000:2900,cccc:9000:2900
ip: 192.168.1.1
password: 18c6!@nkBNK9P!*d8&1Iq2Qt
extend1: abc
extend2: def
save 0 [aaa bbb ccc]
save 1 [eee fff ggg]
`,
		content: `default:
  test: xxxxxx
  connect: '["aaaa","bbbb","cccc"]'
  save: '[["aaa","bbb","ccc"],["eee","fff","ggg"]]'
`,
		extetend_content: `extend1: abc
extend2: def
`,
	},
	//	{
	//		desc: "base test 2",
	//		tmpl: `test: {{ getv "/default/test" }}
	//mul: {{ mul 1 2 }}
	//`,
	//		expected: `test: xxxxxx
	//mul: 2
	//`,
	//		content: `default:
	//  test: xxxxxx
	//`,
	//	},
}

// TestTemplates runs all tests in templateTests
//func TestTemplates(t *testing.T) {
//	for _, tt := range templateTests {
//		ExecuteTestTemplate(tt, t)
//	}
//}

// ExectureTestTemplate builds a TemplateResource based on the toml and tmpl files described
// in the templateTest, writes a config file, and compares the result against the expectation
// in the templateTest.
func ExecuteTestTemplate(tt templateTest, t *testing.T) {
	setupDirectoriesAndFiles(tt, t)
	defer os.RemoveAll("test")

	tr, err := templateResource()
	if err != nil {
		t.Errorf(tt.desc + ": failed to create TemplateResource: " + err.Error())
	}

	tt.updateStore(tr)

	if err := tr.createStageFile(); err != nil {
		t.Errorf(tt.desc + ": failed createStageFile: " + err.Error())
	}

	actual, err := ioutil.ReadFile(tr.StageFile.Name())
	if err != nil {
		t.Errorf(tt.desc + ": failed to read StageFile: " + err.Error())
	}
	switch tt.expected.(type) {
	case string:
		if string(actual) != tt.expected.(string) {
			t.Errorf(fmt.Sprintf("%v: invalid StageFile. Expected %v, actual %v", tt.desc, tt.expected, string(actual)))
		}
	case []string:
		for _, expected := range tt.expected.([]string) {
			if string(actual) == expected {
				break
			}
		}
		t.Errorf(fmt.Sprintf("%v: invalid StageFile. Possible expected values %v, actual %v", tt.desc, tt.expected, string(actual)))
	}
}

// setUpDirectoriesAndFiles creates folders for the toml, tmpl, and output files and
// creates the toml and tmpl files as specified in the templateTest struct.
func setupDirectoriesAndFiles(tt templateTest, t *testing.T) {
	// create confd directory and toml file
	if err := os.MkdirAll("./test/confd", os.ModePerm); err != nil {
		t.Errorf(tt.desc + ": failed to created confd directory: " + err.Error())
	}
	if err := ioutil.WriteFile(tomlFilePath, []byte(tt.toml), os.ModePerm); err != nil {
		t.Errorf(tt.desc + ": failed to write toml file: " + err.Error())
	}
	// create templates directory and tmpl file
	if err := os.MkdirAll("./test/templates", os.ModePerm); err != nil {
		t.Errorf(tt.desc + ": failed to create template directory: " + err.Error())
	}
	if err := ioutil.WriteFile(tmplFilePath, []byte(tt.tmpl), os.ModePerm); err != nil {
		t.Errorf(tt.desc + ": failed to write toml file: " + err.Error())
	}
	// create tmp directory for output
	if err := os.MkdirAll("./test/tmp", os.ModePerm); err != nil {
		t.Errorf(tt.desc + ": failed to create tmp directory: " + err.Error())
	}
}

// templateResource creates a templateResource for creating a config file
func templateResource() (*TemplateResource, error) {
	backendConf := backends.Config{
		Backend: "env"}
	client, err := backends.New(backendConf)
	if err != nil {
		return nil, err
	}

	config := Config{
		StoreClient: client, // not used but must be set
		//TemplateDir: "./test/templates",
	}

	tr, err := NewTemplateResource(config)
	if err != nil {
		return nil, err
	}
	tr.Dest = "./test/tmp/test.conf"
	tr.FileMode = 0666
	return tr, nil
}
