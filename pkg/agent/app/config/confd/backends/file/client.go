package file

import (
	"gopkg.in/yaml.v2"
	"os"
	"path"
	"strconv"
)

// Client provides a shell for the yaml client
type Client struct {
	filepath string
}

func NewFileClient(filepath string) *Client {
	return &Client{
		filepath: filepath,
	}
}

func readFile(path string, vars map[string]string) error {
	yamlMap := make(map[interface{}]interface{})
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &yamlMap)
	if err != nil {
		return err
	}

	err = nodeWalk(yamlMap, "/", vars)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetValues() (map[string]string, error) {

	vars := make(map[string]string)

	err := readFile(c.filepath, vars)
	if err != nil {
		return nil, err
	}

	return vars, nil
}

// nodeWalk recursively descends nodes, updating vars.
func nodeWalk(node interface{}, key string, vars map[string]string) error {
	switch node.(type) {
	case []interface{}:
		for i, j := range node.([]interface{}) {
			key := path.Join(key, strconv.Itoa(i))
			nodeWalk(j, key, vars)
		}
	case map[interface{}]interface{}:
		for k, v := range node.(map[interface{}]interface{}) {
			key := path.Join(key, k.(string))
			nodeWalk(v, key, vars)
		}
	case string:
		vars[key] = node.(string)
	case int:
		vars[key] = strconv.Itoa(node.(int))
	case bool:
		vars[key] = strconv.FormatBool(node.(bool))
	case float64:
		vars[key] = strconv.FormatFloat(node.(float64), 'f', -1, 64)
	}
	return nil
}
