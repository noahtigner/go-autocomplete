package sets

import "math/bits"

// create a mask selecting bit n
// uint64(1) << n

// turn bit 3 on
// x = x | (uint64(1) << 3)
// or
// x |= uint64(1) << 3

// check if bit 3 is set
// mask := uint64(1) << 3
// result := x & mask
// or
// result := x & (uint64(1) << 3) != 0

// clear the lowest set bit
// word &= word - 1

type BitSet struct {
	words []uint64
}

func NewBitSet(size int) *BitSet {
	// Integer division rounded upward
	wordCount := (size + 63) / 64

	return &BitSet{
		words: make([]uint64, wordCount),
	}
}

func (b *BitSet) Set(id int) {
	word := id / 64
	bit := id % 64

	b.words[word] |= uint64(1) << bit
}

func (b *BitSet) Contains(id int) bool {
	word := id / 64
	bit := id % 64

	mask := uint64(1) << bit

	return b.words[word]&mask != 0
}

func (b *BitSet) PopCount() int {
	count := 0

	for _, word := range b.words {
		count += bits.OnesCount64(word)
	}

	return count
}

func (b *BitSet) GetSetIds() []int {
	var result []int

	for wordIndex, word := range b.words {
		for word != 0 {
			// Find the lowest set bit
			lowestSetBit := bits.TrailingZeros64(word)

			// Convert the lowest set bit back to an ID
			id := wordIndex*64 + lowestSetBit
			result = append(result, id)

			// Unset the lowest set bit
			word &= word - 1
		}
	}

	return result
}

func ForEachIntersection(bitSets []*BitSet, visit func(slot int)) int {
	if len(bitSets) == 0 {
		return 0
	}

	total := 0
	for wordIndex, word := range bitSets[0].words {
		// Intersect word across all bitSets
		for _, bitSet := range bitSets[1:] {
			word &= bitSet.words[wordIndex]
		}

		// Increment total w/ intersected word's popcount
		total += bits.OnesCount64(word)

		if visit == nil {
			continue
		}

		// Iterate over each active slot in the intersected word
		for word != 0 {
			// Find the lowest set bit
			bitOffset := bits.TrailingZeros64(word)

			// Call the callback with the slot ID
			slot := wordIndex*64 + bitOffset
			visit(slot)

			// Unset the lowest set bit
			word &= word - 1
		}
	}

	return total
}
