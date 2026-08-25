package sets

import (
	"fmt"
	"slices"
	"testing"
)

func testBitSet(ids ...int) *BitSet {
	bitSet := NewBitSet(130)
	for _, id := range ids {
		bitSet.Set(id)
	}
	return bitSet
}

func TestNewBitSet(t *testing.T) {
	tests := []struct {
		size    int
		wantLen int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{64, 1},
		{65, 2},
		{128, 2},
		{12_699_818, 198_435},
		{13_000_000, 203_125},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("NewBitSet(%d)", tt.size), func(t *testing.T) {
			bs := NewBitSet(tt.size)
			gotLen := len(bs.words)
			if gotLen != tt.wantLen {
				t.Errorf("got %d, want %d", gotLen, tt.wantLen)
			}
		})
	}
}

func TestBitSetSet(t *testing.T) {
	tests := []struct {
		testName string
		size     int
		ids      []int
		want     []uint64
	}{
		{"Sets bit zero", 64, []int{0}, []uint64{1}},
		{"Sets final bit of first word", 64, []int{63}, []uint64{uint64(1) << 63}},
		{"Sets first bit of second word", 128, []int{64}, []uint64{0, 1}},
		{"Sets final bit of second word", 128, []int{127}, []uint64{0, uint64(1) << 63}},
		{"Preserves previously set bits", 128, []int{0, 64, 91}, []uint64{1, 1 | (uint64(1) << 27)}},
		{"Setting a bit twice is idempotent", 64, []int{12, 12}, []uint64{uint64(1) << 12}},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			bitSet := NewBitSet(tt.size)

			for _, id := range tt.ids {
				bitSet.Set(id)
			}

			for i, want := range tt.want {
				if got := bitSet.words[i]; got != want {
					t.Errorf("words[%d] = %064b, want %064b", i, got, want)
				}
			}
		})
	}
}

func TestBitSetContains(t *testing.T) {
	tests := []struct {
		testName string
		size     int
		setIDs   []int
		id       int
		want     bool
	}{
		{"Returns false for unset bit", 64, nil, 0, false},
		{"Returns true for set bit", 64, []int{12}, 12, true},
		{"Does not match an adjacent set bit", 64, []int{12}, 13, false},
		{"Finds final bit in first word", 128, []int{63}, 63, true},
		{"Finds first bit in second word", 128, []int{64}, 64, true},
		{"Does not confuse bits in separate words", 128, []int{64}, 0, false},
		{"Finds final bit in second word", 128, []int{127}, 127, true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			bitSet := NewBitSet(tt.size)

			for _, id := range tt.setIDs {
				bitSet.Set(id)
			}

			if got := bitSet.Contains(tt.id); got != tt.want {
				t.Errorf("Contains(%d) = %t, want %t", tt.id, got, tt.want)
			}
		})
	}
}

func TestBitSetPopCount(t *testing.T) {
	tests := []struct {
		name string
		ids  []int
		want int
	}{
		{name: "empty bitmap"},
		{name: "one word", ids: []int{0, 2, 63}, want: 3},
		{name: "multiple words", ids: []int{0, 63, 64, 127, 129}, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testBitSet(tt.ids...).PopCount(); got != tt.want {
				t.Errorf("PopCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBitSetGetSetIds(t *testing.T) {
	tests := []struct {
		name string
		ids  []int
		want []int
	}{
		{name: "empty bitmap"},
		{name: "ascending within a word", ids: []int{63, 2, 0}, want: []int{0, 2, 63}},
		{name: "ascending across words", ids: []int{129, 64, 127, 0}, want: []int{0, 64, 127, 129}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testBitSet(tt.ids...).GetSetIds(); !slices.Equal(got, tt.want) {
				t.Errorf("GetSetIds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForEachIntersection(t *testing.T) {
	tests := []struct {
		name      string
		inputs    [][]int
		callVisit bool
		wantTotal int
		wantSlots []int
	}{
		{name: "empty input", callVisit: true},
		{
			name:      "single bitmap across word boundaries",
			inputs:    [][]int{{0, 63, 64, 129}},
			callVisit: true,
			wantTotal: 4,
			wantSlots: []int{0, 63, 64, 129},
		},
		{
			name: "three bitmap intersection across word boundaries",
			inputs: [][]int{
				{0, 2, 63, 64, 127, 129},
				{2, 63, 64, 65, 127, 129},
				{2, 63, 64, 127},
			},
			callVisit: true,
			wantTotal: 4,
			wantSlots: []int{2, 63, 64, 127},
		},
		{
			name:      "disjoint bitmaps",
			inputs:    [][]int{{1, 64}, {2, 65}},
			callVisit: true,
		},
		{
			name:      "nil visitor still counts",
			inputs:    [][]int{{0, 64, 129}, {0, 64, 129}},
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := make([]*BitSet, len(tt.inputs))
			originalWords := make([][]uint64, len(tt.inputs))
			for i, ids := range tt.inputs {
				inputs[i] = testBitSet(ids...)
				originalWords[i] = slices.Clone(inputs[i].words)
			}

			var gotSlots []int
			var visit func(int)
			if tt.callVisit {
				visit = func(slot int) {
					gotSlots = append(gotSlots, slot)
				}
			}

			if got := ForEachIntersection(inputs, visit); got != tt.wantTotal {
				t.Errorf("total = %d, want %d", got, tt.wantTotal)
			}
			if !slices.Equal(gotSlots, tt.wantSlots) {
				t.Errorf("visited slots = %v, want %v", gotSlots, tt.wantSlots)
			}
			for i, input := range inputs {
				if !slices.Equal(input.words, originalWords[i]) {
					t.Errorf("input %d was mutated", i)
				}
			}
		})
	}
}

func TestBitFieldSet(t *testing.T) {
	tests := []struct {
		name    string
		indexes []int
		want    uint32
	}{
		{"Control", []int{}, uint32(0)},
		{"Single low bit set", []int{0}, uint32(1)},
		{"Single high bit set", []int{31}, uint32(1) << 31},
		{"Overflow wraps to 0 instead of panicking", []int{1234}, uint32(0)},
		{"Preserves previously set bits", []int{0, 7}, uint32(1 | (uint32(1) << 7))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf := BitField(0)
			for _, id := range tt.indexes {
				bf.Set(id)
			}

			want := BitField(tt.want)
			if want != bf {
				t.Errorf("got %032b, want %032b (%d)", bf, want, want)
			}
		})
	}
}

func TestBitFieldContains(t *testing.T) {
	tests := []struct {
		testName string
		setIDs   []int
		id       int
		want     bool
	}{
		{"Returns false for unset bit", nil, 0, false},
		{"Returns true for set bit", []int{12}, 12, true},
		{"Does not match an adjacent set bit", []int{12}, 13, false},
		{"Finds final bit in", []int{31}, 31, true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			bf := BitField(0)

			for _, id := range tt.setIDs {
				bf.Set(id)
			}

			if got := bf.Contains(tt.id); got != tt.want {
				t.Errorf("%032b.Contains(%d) = %t, want %t", bf, tt.id, got, tt.want)
			}
		})
	}
}

func TestBitFieldHasIntersection(t *testing.T) {
	tests := []struct {
		testName string
		b1Vals   []int
		b2Vals   []int
		want     bool
	}{
		{"Both unset", []int{}, []int{}, false},
		{"Both set", []int{0}, []int{0}, true},
		{"Left set only", []int{0}, []int{}, false},
		{"No overlap", []int{1, 3}, []int{2}, false},
		{"Some overlap", []int{1, 3}, []int{2, 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			b1 := BitField(0)
			b2 := BitField(0)

			for _, v := range tt.b1Vals {
				b1.Set(v)
			}
			for _, v := range tt.b2Vals {
				b2.Set(v)
			}

			got := b1.HasIntersection(b2)

			if got != tt.want {
				t.Errorf("got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBitFieldGetSetIndexes(t *testing.T) {
	tests := []struct {
		testName string
		setVals  []int
	}{
		{"None set", []int{}},
		{"Single bit set", []int{0}},
		{"Multiple bits set", []int{1, 9, 16}},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			bf := BitField(0)
			for _, v := range tt.setVals {
				bf.Set(v)
			}

			got := bf.GetSetIndexes()

			if !slices.Equal(got, tt.setVals) {
				t.Errorf("got %v, want %v", got, tt.setVals)
			}
		})
	}
}

func TestBitFieldIntersection(t *testing.T) {
	tests := []struct {
		testName string
		b1Vals   []int
		b2Vals   []int
		wantVals []int
	}{
		{"Both unset", []int{}, []int{}, []int{}},
		{"Both set", []int{0}, []int{0}, []int{0}},
		{"Left set only", []int{0}, []int{}, []int{}},
		{"No overlap", []int{1, 3}, []int{2}, []int{}},
		{"Some overlap", []int{1, 3}, []int{2, 3}, []int{3}},
		{"Preserves high bit", []int{0, 31}, []int{1, 31}, []int{31}},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			b1 := BitField(0)
			b2 := BitField(0)
			want := BitField(0)

			for _, v := range tt.b1Vals {
				b1.Set(v)
			}
			for _, v := range tt.b2Vals {
				b2.Set(v)
			}
			for _, v := range tt.wantVals {
				want.Set(v)
			}

			if got := b1.Intersection(b2); got != want {
				t.Errorf("got %032b, want %032b", got, want)
			}
		})
	}
}
