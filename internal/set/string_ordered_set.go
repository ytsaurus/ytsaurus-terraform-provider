package set

import (
	"sort"
	"strings"
)

type StringOrderedSet map[string]int

func ToStringSet(keys []string) StringOrderedSet {
	s := make(StringOrderedSet)
	for i, k := range keys {
		s[k] = i
	}
	return s
}

func (a StringOrderedSet) Contains(k string) bool {
	_, ok := a[k]
	return ok
}

func (a StringOrderedSet) Difference(b StringOrderedSet) []string {
	var r []string
	for k := range a {
		if !b.Contains(k) {
			r = append(r, k)
		}
	}
	return r
}

func (a StringOrderedSet) IsEqual(b StringOrderedSet) bool {
	if len(a) != len(b) {
		return false
	}
	for k, _ := range a {
		if !b.Contains(k) {
			return false
		}
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

func (a StringOrderedSet) String() string {
	var keys []string
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
