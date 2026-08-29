package kubernetes

func neatObject(obj map[string]any, neat bool) map[string]any {
	if !neat {
		return obj
	}
	return stripManagedFields(obj)
}

func neatList(envelope map[string]any, neat bool) map[string]any {
	if !neat {
		return envelope
	}
	return stripManagedFieldsFromList(envelope)
}

func stripManagedFields(obj map[string]any) map[string]any {
	metadata, ok := obj["metadata"].(map[string]any)
	if !ok {
		return obj
	}
	if _, has := metadata["managedFields"]; !has {
		return obj
	}
	newMetadata := make(map[string]any, len(metadata)-1)
	for k, v := range metadata {
		if k == "managedFields" {
			continue
		}
		newMetadata[k] = v
	}
	newObj := make(map[string]any, len(obj))
	for k, v := range obj {
		newObj[k] = v
	}
	newObj["metadata"] = newMetadata
	return newObj
}

func stripManagedFieldsFromList(envelope map[string]any) map[string]any {
	items, ok := envelope["items"].([]any)
	if !ok {
		return envelope
	}
	stripped := make([]any, len(items))
	for i, item := range items {
		if obj, ok := item.(map[string]any); ok {
			stripped[i] = stripManagedFields(obj)
		} else {
			stripped[i] = item
		}
	}
	out := make(map[string]any, len(envelope))
	for k, v := range envelope {
		out[k] = v
	}
	out["items"] = stripped
	return out
}
