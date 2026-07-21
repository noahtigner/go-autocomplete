package dataStructures

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

func (s Set[T]) Remove(item T) {
	delete(s, item)
}

func (s Set[T]) Contains(item T) bool {
	_, exists := s[item]
	return exists
}

func (s Set[T]) Get(item T) (T, bool) {
	return item, s.Contains(item)
}

func (s Set[T]) Intersection(otherSets []Set[T]) Set[T] {
	if len(otherSets) == 0 {
		return s
	}

	intersection := s
	for _, otherSet := range otherSets {
		for key := range intersection {
			if !otherSet.Contains(key) {
				intersection.Remove(key)
			}
		}
	}
	return intersection
}
