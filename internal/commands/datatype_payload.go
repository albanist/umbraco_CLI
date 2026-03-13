package commands

import (
	"context"
	"fmt"

	"umbraco-cli/internal/api"
)

func fetchDatatypeObject(ctx context.Context, client *api.Client, id string) (map[string]any, error) {
	result, err := client.Get(ctx, fmt.Sprintf("%s/%s", dataTypeLegacyCollectionPath, id), api.RequestOptions{})
	if err != nil {
		return nil, err
	}

	return decodeResult[map[string]any](result)
}

func mergeDatatypePayload(current map[string]any, patch map[string]any) map[string]any {
	merged := cloneObject(current)
	for key, value := range patch {
		if existing, exists := merged[key]; exists {
			merged[key] = mergeDatatypeValue(existing, value)
			continue
		}

		merged[key] = cloneDatatypeValue(value)
	}

	return merged
}

func mergeDatatypeValue(current any, patch any) any {
	currentMap, currentIsMap := current.(map[string]any)
	patchMap, patchIsMap := patch.(map[string]any)
	if currentIsMap && patchIsMap {
		return mergeDatatypePayload(currentMap, patchMap)
	}

	currentArray, currentIsArray := current.([]any)
	patchArray, patchIsArray := patch.([]any)
	if currentIsArray && patchIsArray && isAliasObjectArray(currentArray) && isAliasObjectArray(patchArray) {
		return mergeAliasObjectArrays(currentArray, patchArray)
	}

	return cloneDatatypeValue(patch)
}

func mergeAliasObjectArrays(current []any, patch []any) []any {
	merged := make([]any, 0, len(current)+len(patch))
	patchByAlias := make(map[string]map[string]any, len(patch))
	for _, item := range patch {
		alias, itemMap, ok := aliasObject(item)
		if !ok {
			continue
		}
		patchByAlias[alias] = itemMap
	}

	seen := make(map[string]struct{}, len(patchByAlias))
	for _, item := range current {
		alias, itemMap, ok := aliasObject(item)
		if !ok {
			merged = append(merged, cloneDatatypeValue(item))
			continue
		}

		patchItem, hasPatch := patchByAlias[alias]
		if !hasPatch {
			merged = append(merged, cloneDatatypeValue(itemMap))
			continue
		}

		merged = append(merged, mergeDatatypePayload(itemMap, patchItem))
		seen[alias] = struct{}{}
	}

	for _, item := range patch {
		alias, itemMap, ok := aliasObject(item)
		if !ok {
			merged = append(merged, cloneDatatypeValue(item))
			continue
		}
		if _, alreadyMerged := seen[alias]; alreadyMerged {
			continue
		}
		merged = append(merged, cloneDatatypeValue(itemMap))
	}

	return merged
}

func isAliasObjectArray(items []any) bool {
	for _, item := range items {
		if _, _, ok := aliasObject(item); !ok {
			return false
		}
	}
	return true
}

func aliasObject(item any) (string, map[string]any, bool) {
	itemMap, ok := item.(map[string]any)
	if !ok {
		return "", nil, false
	}
	alias, ok := itemMap["alias"].(string)
	if !ok || alias == "" {
		return "", nil, false
	}
	return alias, itemMap, true
}

func cloneObject(input map[string]any) map[string]any {
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = cloneDatatypeValue(value)
	}
	return cloned
}

func cloneDatatypeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneObject(typed)
	case []any:
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = cloneDatatypeValue(item)
		}
		return cloned
	default:
		return typed
	}
}
