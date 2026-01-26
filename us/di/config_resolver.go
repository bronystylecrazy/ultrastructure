package di

import (
	"fmt"
	"reflect"
)

type configTarget interface {
	configType() reflect.Type
	configTarget() (path string, key string, cfg configConfig, ok bool)
}

type configSource interface {
	configSource() (path string, cfg configConfig, ok bool)
}

func (n configNode[T]) configType() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (n configNode[T]) configTarget() (string, string, configConfig, bool) {
	cfg, _, _, _, err := parseConfigOptionsWithWatch(n.opts)
	if err != nil {
		return "", "", configConfig{}, false
	}
	path, key := resolveConfigTarget(n.pathOrKey)
	if path == "" && key == "" {
		return "", "", configConfig{}, false
	}
	return path, key, cfg, true
}

func (n configFileNode) configSource() (string, configConfig, bool) {
	cfg, _, _, err := parseConfigOptions(n.opts)
	if err != nil {
		return "", configConfig{}, false
	}
	if n.path == "" {
		return "", configConfig{}, false
	}
	return n.path, cfg, true
}

func buildConfigResolver(nodes []Node) func(reflect.Type) (any, error) {
	targets, source := collectConfigTargets(nodes)
	return func(t reflect.Type) (any, error) {
		tgt, ok := targets[t]
		if !ok {
			return nil, configTypeNotFoundError{typ: t}
		}
		path, key, cfg := tgt.path, tgt.key, tgt.cfg
		if !hasConfigSource(cfg) && path == "" {
			if source.ok {
				path = source.path
				cfg = source.cfg
			}
		}
		if path == "" && !hasConfigSource(cfg) {
			return nil, fmt.Errorf("no config source for %s", t)
		}
		v, err := loadViper(cfg, path)
		if err != nil {
			return nil, err
		}
		ptr := reflect.New(t)
		if key == "" {
			if err := v.Unmarshal(ptr.Interface()); err != nil {
				return nil, err
			}
			return ptr.Elem().Interface(), nil
		}
		if err := v.UnmarshalKey(key, ptr.Interface()); err != nil {
			return nil, err
		}
		return ptr.Elem().Interface(), nil
	}
}

type configTargetEntry struct {
	path string
	key  string
	cfg  configConfig
}

type configSourceEntry struct {
	path string
	cfg  configConfig
	ok   bool
}

type configTypeNotFoundError struct {
	typ reflect.Type
}

func (e configTypeNotFoundError) Error() string {
	return fmt.Sprintf("config type %s not found", e.typ)
}

func collectConfigTargets(nodes []Node) (map[reflect.Type]configTargetEntry, configSourceEntry) {
	targets := make(map[reflect.Type]configTargetEntry)
	var source configSourceEntry
	walkNodes(nodes, func(n Node) {
		switch v := n.(type) {
		case configTarget:
			typ := v.configType()
			if _, exists := targets[typ]; exists {
				return
			}
			path, key, cfg, ok := v.configTarget()
			if !ok {
				return
			}
			targets[typ] = configTargetEntry{path: path, key: key, cfg: cfg}
		case configSource:
			if source.ok {
				return
			}
			path, cfg, ok := v.configSource()
			if !ok {
				return
			}
			source = configSourceEntry{path: path, cfg: cfg, ok: true}
		}
	})
	return targets, source
}

func attachConfigResolvers(nodes []Node, resolver func(reflect.Type) (any, error)) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case conditionalNode:
			v.resolver = resolver
			v.nodes = attachConfigResolvers(v.nodes, resolver)
			out[i] = v
		case moduleNode:
			v.nodes = attachConfigResolvers(v.nodes, resolver)
			out[i] = v
		case optionsNode:
			v.nodes = attachConfigResolvers(v.nodes, resolver)
			out[i] = v
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for idx, c := range v.cases {
				c.resolver = resolver
				c.nodes = attachConfigResolvers(c.nodes, resolver)
				cases[idx] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: attachConfigResolvers(v.defaultCase.nodes, resolver)}
			out[i] = v
		default:
			out[i] = n
		}
	}
	return out
}

func walkNodes(nodes []Node, fn func(Node)) {
	for _, n := range nodes {
		fn(n)
		switch v := n.(type) {
		case moduleNode:
			walkNodes(v.nodes, fn)
		case optionsNode:
			walkNodes(v.nodes, fn)
		case conditionalNode:
			walkNodes(v.nodes, fn)
		case switchNode:
			for _, c := range v.cases {
				walkNodes(c.nodes, fn)
			}
			walkNodes(v.defaultCase.nodes, fn)
		}
	}
}

func collectGlobalConfigWatch(nodes []Node) (configWatchConfig, bool) {
	var cfg configWatchConfig
	found := false
	walkNodes(nodes, func(n Node) {
		if w, ok := n.(configWatchAll); ok {
			found = true
			w.applyWatch(&cfg)
		}
	})
	return cfg, found
}

func applyGlobalConfigWatch(nodes []Node, cfg configWatchConfig) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		switch v := n.(type) {
		case configWatchAll:
			out[i] = n
		case moduleNode:
			v.nodes = applyGlobalConfigWatch(v.nodes, cfg)
			out[i] = v
		case optionsNode:
			v.nodes = applyGlobalConfigWatch(v.nodes, cfg)
			out[i] = v
		case conditionalNode:
			v.nodes = applyGlobalConfigWatch(v.nodes, cfg)
			out[i] = v
		case switchNode:
			cases := make([]caseNode, len(v.cases))
			for idx, c := range v.cases {
				c.nodes = applyGlobalConfigWatch(c.nodes, cfg)
				cases[idx] = c
			}
			v.cases = cases
			v.defaultCase = switchDefaultNode{nodes: applyGlobalConfigWatch(v.defaultCase.nodes, cfg)}
			out[i] = v
		default:
			if wc, ok := n.(interface{ withConfigWatch(configWatchConfig) Node }); ok {
				out[i] = wc.withConfigWatch(cfg)
				continue
			}
			out[i] = n
		}
	}
	return out
}
