package template

import "encoding/json"

func fromJSON(s string) []map[string]any {
	var result []map[string]any
	_ = json.Unmarshal([]byte(s), &result)
	return result
}
