package kubernetes

import "strings"

func projectObject(obj map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return obj
	}
	result := map[string]any{}
	for _, field := range fields {
		path := strings.Split(field, ".")
		value, ok := getPath(obj, path)
		if !ok {
			continue
		}
		setPath(result, path, value)
	}
	return result
}

func projectList(envelope map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return envelope
	}
	items, ok := envelope["items"].([]any)
	if !ok {
		return envelope
	}
	projected := make([]any, len(items))
	for i, item := range items {
		if obj, ok := item.(map[string]any); ok {
			projected[i] = projectObject(obj, fields)
		} else {
			projected[i] = item
		}
	}
	out := make(map[string]any, len(envelope))
	for k, v := range envelope {
		out[k] = v
	}
	out["items"] = projected
	return out
}

func getPath(obj map[string]any, path []string) (any, bool) {
	var cur any = obj
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func setPath(obj map[string]any, path []string, value any) {
	cur := obj
	for i, seg := range path {
		if i == len(path)-1 {
			cur[seg] = value
			return
		}
		next, ok := cur[seg].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[seg] = next
		}
		cur = next
	}
}
