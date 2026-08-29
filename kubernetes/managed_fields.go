package kubernetes

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
