package di

func ConvertAnys(nodes []Node) []any {
	// Convert []Node to []any for APIs that accept mixed node/option inputs.
	anyExtends := make([]any, len(nodes))
	for i, ext := range nodes {
		anyExtends[i] = ext
	}
	return anyExtends
}
