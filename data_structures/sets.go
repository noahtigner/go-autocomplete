package dataStructures

import "maps"

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

func (s Set[T]) ToSlice() []T {
	slice := make([]T, len(s))
	i := 0
	for item := range s {
		slice[i] = item
		i += 1
	}
	return slice
}

// suboptimal; use the next implementation
func (s Set[T]) Intersection(otherSets []Set[T]) Set[T] {
	if len(otherSets) == 0 {
		return s
	}

	intersection := s
	for _, otherSet := range otherSets {
		for key := range intersection {
			if !otherSet.Contains(key) {
				intersection.Remove(key)
				break
			}
		}
	}
	return intersection
}

// optimal approach
func Intersection[T comparable](sets []Set[T]) Set[T] {
	if len(sets) == 0 {
		return NewSet[T]()
	}
	if len(sets) == 1 {
		return sets[0]
	}

	// find the smallest set
	minSet := 0
	for i := 1; i < len(sets); i += 1 {
		if len(sets[i]) < len(sets[minSet]) {
			minSet = i
		}
	}

	// intersect the smallest set with the rest, keeping the intermediate set minimized
	result := maps.Clone(sets[minSet])
	for item := range result {
		for i, otherSet := range sets {
			if i == minSet {
				continue
			}
			if !otherSet.Contains(item) {
				result.Remove(item)
				break
			}
		}
	}

	return result
}

func Unique[T comparable](items []T) []T {
	seen := NewSet[T]()
	result := make([]T, 0, len(items))

	for _, item := range items {
		if !seen.Contains(item) {
			seen.Add(item)
			result = append(result, item)
		}
	}

	return result
}
