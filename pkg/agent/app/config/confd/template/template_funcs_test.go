package template

import (
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	testDir = "./test"
)

type templateTest struct {
	desc             string      // description of the test (for helpful errors)
	tmpl             string      // template file contents
	content          string      // value contents
	extetend_content string      // value extend contents
	expected         interface{} // expected generated file contents
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
extend1: {{ getv "/extend1" }}
extend2: {{ getv "/extend2" }}
{{- range $index,$value := jsonArray (getv "/default/save")}}
save {{ $index }} {{ $value }}
{{- end }}
{{- $key := "key" }}
{{- $count := 6 }}
{{- range $i := seq 0 (sub $count 1) }}
{{ $key }}-{{ $i }}
{{- end }}
`,
		expected: `test: xxxxxx
connect: aaaa:9000:2900,bbbb:9000:2900,cccc:9000:2900
ip: 192.168.1.1
extend1: abc
extend2: def
save 0 [aaa bbb ccc]
save 1 [eee fff ggg]
key-0
key-1
key-2
key-3
key-4
key-5
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
func TestTemplates(t *testing.T) {
	for _, tt := range templateTests {
		ExecuteTestTemplate(tt, t)
	}
}

func ExecuteTestTemplate(tt templateTest, t *testing.T) {
	if err := os.MkdirAll(testDir, os.ModePerm); err != nil {
		t.Error(tt.desc + ": failed to create template directory: " + err.Error())
	}

	//defer os.RemoveAll(testDir)

	if err := ioutil.WriteFile(filepath.Join(testDir, "test.tmpl"), []byte(tt.tmpl), os.ModePerm); err != nil {
		t.Error(tt.desc + ": failed to write template file: " + err.Error())
	}

	backendConf := backends.Config{
		Backend:  "content",
		Contents: []string{tt.content, tt.extetend_content},
	}
	client, err := backends.New(backendConf)
	if err != nil {
		t.Error(tt.desc + ": create backends client failed: " + err.Error())
	}

	config := Config{
		StoreClient:  client,
		TemplateFile: filepath.Join(testDir, "test.tmpl"),
		DestFile:     filepath.Join(testDir, "test.conf"),
	}

	t.Setenv("HOST_IP", "192.168.1.1")
	err = Process(config)
	if err != nil {
		t.Error(tt.desc + ": generate failed: " + err.Error())
	}

	actual, err := os.ReadFile(filepath.Join(testDir, "test.conf"))
	if err != nil {
		t.Error(tt.desc + ": failed to read StageFile: " + err.Error())
	}
	t.Log(string(actual))

	if string(actual) != tt.expected.(string) {
		t.Error(fmt.Sprintf("%v: invalid StageFile.\nExpected:\n%vActual:\n%v", tt.desc, tt.expected, string(actual)))
	}
}
