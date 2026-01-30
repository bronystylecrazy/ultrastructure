package di

import "sort"

func collectGlobalTagSets(nodes []Node) ([]tagSet, []decorateEntry, error) {
	return collectScope(nodes)
}

func collectScope(nodes []Node) ([]tagSet, []decorateEntry, error) {
	var (
		out          []tagSet
		decorators   []decorateEntry
		replacements []replaceSpec
		pos          int
		localDecs    []decorateNode
		provideItems []provideItem
	)

	for _, n := range nodes {
		switch v := n.(type) {
		case provideNode:
			cfg, decs, _, err := parseBindOptions(v.opts)
			if err != nil {
				return nil, nil, err
			}
			_, tagSets, err := buildProvideSpec(cfg, v.constructor, nil)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			provideItems = append(provideItems, provideItem{pos: pos, node: v, tagSets: tagSets})
			for _, dec := range decs {
				decorators = append(decorators, decorateEntry{
					dec:      dec,
					tagSets:  tagSets,
					position: pos,
				})
				pos++
			}
			pos++
		case supplyNode:
			cfg, decs, _, err := parseBindOptions(v.opts)
			if err != nil {
				return nil, nil, err
			}
			_, tagSets, err := buildProvideSpec(cfg, nil, v.value)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			provideItems = append(provideItems, provideItem{pos: pos, node: v, tagSets: tagSets})
			for _, dec := range decs {
				decorators = append(decorators, decorateEntry{
					dec:      dec,
					tagSets:  tagSets,
					position: pos,
				})
				pos++
			}
			pos++
		case replaceNode:
			spec, err := buildReplaceSpec(v, pos)
			if err != nil {
				return nil, nil, err
			}
			replacements = append(replacements, spec)
			pos++
		case defaultNode:
			spec, err := buildDefaultSpec(v, pos)
			if err != nil {
				return nil, nil, err
			}
			replacements = append(replacements, spec)
			pos++
		case moduleNode:
			tagSets, decs, err := collectScope(v.nodes)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			decorators = append(decorators, decs...)
		case optionsNode:
			tagSets, decs, err := collectScope(v.nodes)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			decorators = append(decorators, decs...)
		case switchNode:
			selected, err := v.selectNodes()
			if err != nil {
				return nil, nil, err
			}
			if len(selected) == 0 {
				continue
			}
			tagSets, decs, err := collectScope(selected)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			decorators = append(decorators, decs...)
		case conditionalNode:
			ok, err := v.eval()
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				continue
			}
			tagSets, decs, err := collectScope(v.nodes)
			if err != nil {
				return nil, nil, err
			}
			out = append(out, tagSets...)
			decorators = append(decorators, decs...)
		case decorateNode:
			localDecs = append(localDecs, v)
		}
	}

	if len(replacements) > 0 {
		// Recompute tag sets after replacements so decorators target the final graph.
		filtered := make([]provideItem, 0, len(provideItems))
		for _, p := range provideItems {
			if !matchesReplace(p.tagSets, replacements) {
				filtered = append(filtered, p)
			}
		}
		provideItems = filtered
		for _, r := range replacements {
			if r.node != nil {
				provideItems = append(provideItems, provideItem{pos: r.pos, node: r.node, tagSets: r.tagSets})
			}
		}
		sort.SliceStable(provideItems, func(i, j int) bool {
			return provideItems[i].pos < provideItems[j].pos
		})
		out = nil
		for _, p := range provideItems {
			out = append(out, p.tagSets...)
		}
	}

	for _, dec := range localDecs {
		decorators = append(decorators, decorateEntry{
			dec:      dec,
			tagSets:  out,
			position: pos,
		})
		pos++
	}

	return out, decorators, nil
}
