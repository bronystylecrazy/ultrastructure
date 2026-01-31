package di

type basicThing struct {
	value string
}

func newBasicThing() *basicThing {
	return &basicThing{value: "provided"}
}

type namedThing struct {
	value string
}

func newNamedPrimary() *namedThing {
	return &namedThing{value: "primary"}
}

func newNamedSecondary() *namedThing {
	return &namedThing{value: "secondary"}
}

type depThing struct {
	id int
}

type depCollector struct {
	ids []int
}

func newDepCollector(deps []depThing) *depCollector {
	out := make([]int, 0, len(deps))
	for _, d := range deps {
		out = append(out, d.id)
	}
	return &depCollector{ids: out}
}
