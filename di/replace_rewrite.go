package di

import (
	"reflect"
	"strings"
)

func rewriteInvokeWithTags(node invokeNode, activeTags map[string]tagSet) (invokeNode, bool, error) {
	if len(activeTags) == 0 {
		return node, false, nil
	}
	fnType := reflect.TypeOf(node.function)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return node, false, nil
	}
	numIn := fnType.NumIn()
	if numIn == 0 {
		return node, false, nil
	}
	var cfg paramConfig
	if err := applyParamOptions(node.opts, &cfg); err != nil {
		return node, false, err
	}
	tags := buildParamTags(numIn, cfg.tags)
	tags, changed := rewriteParamTags(fnType, tags, activeTags)
	if !changed {
		return node, false, nil
	}
	return invokeNode{
		function:          node.function,
		opts:              node.opts,
		paramTagsOverride: tags,
	}, true, nil
}

func rewriteProvideWithTags(node provideNode, activeTags map[string]tagSet) (provideNode, bool, error) {
	if len(activeTags) == 0 {
		return node, false, nil
	}
	fnType := reflect.TypeOf(node.constructor)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return node, false, nil
	}
	numIn := fnType.NumIn()
	if numIn == 0 {
		return node, false, nil
	}
	if node.paramTagsOverride != nil {
		tags := buildParamTags(numIn, node.paramTagsOverride)
		tags, changed := rewriteParamTags(fnType, tags, activeTags)
		if !changed {
			return node, false, nil
		}
		node.paramTagsOverride = tags
		return node, true, nil
	}
	cfg, _, _, err := parseBindOptions(node.opts)
	if err != nil {
		return node, false, err
	}
	tags := buildParamTags(numIn, cfg.paramTags)
	tags, changed := rewriteParamTags(fnType, tags, activeTags)
	if !changed {
		return node, false, nil
	}
	node.paramTagsOverride = tags
	return node, true, nil
}

func buildParamTags(numIn int, src []string) []string {
	tags := make([]string, numIn)
	for i := 0; i < numIn && i < len(src); i++ {
		tags[i] = src[i]
	}
	return tags
}

func rewriteParamTags(fnType reflect.Type, tags []string, activeTags map[string]tagSet) ([]string, bool) {
	changed := false
	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		tag := tags[i]
		name, group := parseTagNameGroup(tag)
		key := fullTagKey(tagSet{typ: paramType, name: name, group: group})
		scoped, ok := activeTags[key]
		if !ok {
			continue
		}
		newTag := rewriteParamTag(tag, scoped)
		if newTag != tag {
			tags[i] = newTag
			changed = true
		}
	}
	return tags, changed
}

func parseTagNameGroup(tag string) (string, string) {
	name, _ := extractTagValue(tag, `name:"`)
	group, _ := extractTagValue(tag, `group:"`)
	return name, group
}

func extractTagValue(tag string, key string) (string, bool) {
	idx := strings.Index(tag, key)
	if idx < 0 {
		return "", false
	}
	start := idx + len(key)
	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return "", false
	}
	end += start
	return tag[start:end], true
}

func rewriteParamTag(tag string, scoped tagSet) string {
	repl := ""
	if scoped.group != "" {
		repl = `group:"` + scoped.group + `"`
	} else if scoped.name != "" {
		repl = `name:"` + scoped.name + `"`
	} else {
		return tag
	}
	if strings.Contains(tag, `name:"`) {
		if updated, ok := replaceTagValue(tag, `name:"`, scoped.name); ok {
			return updated
		}
	}
	if strings.Contains(tag, `group:"`) {
		if updated, ok := replaceTagValue(tag, `group:"`, scoped.group); ok {
			return updated
		}
	}
	if tag == "" {
		return repl
	}
	return tag + " " + repl
}

func replaceTagValue(tag string, key string, val string) (string, bool) {
	idx := strings.Index(tag, key)
	if idx < 0 {
		return "", false
	}
	start := idx + len(key)
	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return "", false
	}
	end += start
	return tag[:start] + val + tag[end:], true
}
