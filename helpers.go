package kgen

import (
	"slices"

	"golang.org/x/exp/constraints"
)

func MapKeysSorted[K constraints.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0)
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
