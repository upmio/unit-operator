package content

import (
	"gopkg.in/yaml.v2"
	"path"
	"strconv"
)

// Client provides a shell for the yaml client
type Client struct {
	contents []string
}

func NewContentClient(contents []string) *Client {
	return &Client{
		contents: contents,
	}
}

func read(data string, vars map[string]string) error {
	yamlMap := make(map[interface{}]interface{})

	if err := yaml.Unmarshal([]byte(data), &yamlMap); err != nil {
		return err
	}

	return nodeWalk(yamlMap, "/", vars)
}

func (k *Client) GetValues() (map[string]string, error) {
	vars := make(map[string]string)

	for _, data := range k.contents {
		if err := read(data, vars); err != nil {
			return vars, err
		}
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
