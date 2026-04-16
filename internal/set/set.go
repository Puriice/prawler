package set

type Set[K comparable] map[K]struct{}

func NewSet[K comparable](init ...K) Set[K] {
	set := make(map[K]struct{}, len(init))

	for _, value := range init {
		set[value] = struct{}{}
	}

	return set
}

func (s *Set[K]) Add(value K) {
	(*s)[value] = struct{}{}
}

func (s *Set[K]) Remove(value K) {
	delete(*s, value)
}

func (s *Set[K]) Contains(value K) bool {
	_, exist := (*s)[value]

	return exist
}
