package sets

import (
	"maps"
	"slices"
	"testing"
)

func TestSets(t *testing.T) {
	s := NewSet[int]()

	for range s {
		t.Errorf("NewSet should init an empty map, got %v", s)
	}

	s.Add(12)
	if _, exists := s[12]; !exists {
		t.Errorf("Add did not properly insert into the map")
	}

	s.Remove(12)
	if _, exists := s[12]; exists {
		t.Errorf("Remove did not properly remove from the map")
	}

	if s.Contains(12) {
		t.Errorf("Contains returned false positive for removed item")
	}

	s.Add(99)
	if !s.Contains(99) {
		t.Errorf("Contains returned false negative for added item")
	}

	s.Add(1234)
	s.Add(23)
	gotAsSlice := s.ToSlice()
	wantAsSlice := []int{99, 1234, 23}
	slices.Sort(gotAsSlice)
	slices.Sort(wantAsSlice)
	if !slices.Equal(wantAsSlice, gotAsSlice) {
		t.Errorf("ToSlice returned unexpected slice, want %v, got %v", wantAsSlice, gotAsSlice)
	}
}

func TestSetIntersection(t *testing.T) {
	emptySet := NewSet[string]()
	set1 := NewSet[string]()
	set1.Add("one")
	set2 := NewSet[string]()
	set2.Add("two")
	set3 := NewSet[string]()
	set3.Add("two")
	set3.Add("three")

	setsOverlap := NewSet[string]()
	setsOverlap.Add("two")

	tests := []struct {
		sliceOfSets []Set[string]
		wantSet     Set[string]
		testName    string
	}{
		{[]Set[string]{}, emptySet, "Empty slice of sets"},
		{[]Set[string]{emptySet, set1}, emptySet, "First slice is empty"},
		{[]Set[string]{set2, emptySet, set1}, emptySet, "Any slice is empty"},
		{[]Set[string]{set1, set2}, emptySet, "No overlap"},
		{[]Set[string]{set1, set2, set3}, emptySet, "Some overlap"},
		{[]Set[string]{set2, set3}, setsOverlap, "SomAlle overlap"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got := Intersection(tt.sliceOfSets)
			if !maps.Equal(tt.wantSet, got) {
				t.Errorf("Intersection returned unexpected set, want %v, got %v", tt.wantSet, got)
			}
		})
	}
}

func TestSliceUnique(t *testing.T) {
	tests := []struct {
		inputs   []string
		want     []string
		testName string
	}{
		{[]string{}, []string{}, "Empty slice"},
		{[]string{"one", "two"}, []string{"one", "two"}, "All unique"},
		{[]string{"one", "two", "two"}, []string{"one", "two"}, "Some unique"},
		{[]string{"two", "two"}, []string{"two"}, "None unique"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			want := tt.want
			got := Unique(tt.inputs)
			slices.Sort(want)
			slices.Sort(got)

			if !slices.Equal(got, want) {
				t.Errorf("Unique returned unexpected slice, want %v, got %v", want, got)
			}
		})
	}
}
