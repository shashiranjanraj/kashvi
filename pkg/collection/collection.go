// Package collection provides generic, functional-style helpers for slices.
// It mirrors Laravel's Collection API — Map, Filter, Reject, First, Last,
// Chunk, GroupBy, Pluck, Unique, SortBy, Each, Reduce, Contains, Flatten.
//
// All functions work with Go generics (go 1.21+).
//
// Usage:
//
//	names := collection.Map(users, func(u models.User) string { return u.Name })
//	admins := collection.Filter(users, func(u models.User) bool { return u.Role == "admin" })
//	grouped := collection.GroupBy(users, func(u models.User) string { return u.Role })
package collection

import "sort"

// Map transforms each element of slice s using fn.
func Map[T, R any](s []T, fn func(T) R) []R {
	out := make([]R, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}
	return out
}

// Filter returns elements of s for which fn returns true.
func Filter[T any](s []T, fn func(T) bool) []T {
	var out []T
	for _, v := range s {
		if fn(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reject returns elements of s for which fn returns false (inverse of Filter).
func Reject[T any](s []T, fn func(T) bool) []T {
	return Filter(s, func(v T) bool { return !fn(v) })
}

// Each calls fn for every element (for side-effects). Returns s unchanged.
func Each[T any](s []T, fn func(T)) []T {
	for _, v := range s {
		fn(v)
	}
	return s
}

// First returns the first element matching fn, or (zero, false).
func First[T any](s []T, fn func(T) bool) (T, bool) {
	for _, v := range s {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Last returns the last element matching fn, or (zero, false).
func Last[T any](s []T, fn func(T) bool) (T, bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if fn(s[i]) {
			return s[i], true
		}
	}
	var zero T
	return zero, false
}

// Contains reports whether any element of s satisfies fn.
func Contains[T any](s []T, fn func(T) bool) bool {
	_, ok := First(s, fn)
	return ok
}

// GroupBy partitions s into a map keyed by the string returned by fn.
func GroupBy[T any](s []T, fn func(T) string) map[string][]T {
	out := make(map[string][]T)
	for _, v := range s {
		k := fn(v)
		out[k] = append(out[k], v)
	}
	return out
}

// Pluck extracts a single field from every element.
func Pluck[T, R any](s []T, fn func(T) R) []R {
	return Map(s, fn)
}

// Unique returns s with duplicate elements removed (O(n) via map).
// T must be comparable.
func Unique[T comparable](s []T) []T {
	seen := make(map[T]struct{}, len(s))
	var out []T
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// UniqueBy removes duplicates using a key extracted by fn.
func UniqueBy[T any, K comparable](s []T, fn func(T) K) []T {
	seen := make(map[K]struct{}, len(s))
	var out []T
	for _, v := range s {
		k := fn(v)
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Chunk splits s into slices of at most size n.
func Chunk[T any](s []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	var out [][]T
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

// SortBy sorts s in-place using a key extracted by fn (ascending).
// fn must return a comparable, orderable type — use string or numeric helper below.
func SortBy[T any](s []T, less func(a, b T) bool) []T {
	sort.Slice(s, func(i, j int) bool { return less(s[i], s[j]) })
	return s
}

// Reduce folds s into a single value using fn, starting with initial.
func Reduce[T, R any](s []T, initial R, fn func(carry R, item T) R) R {
	carry := initial
	for _, v := range s {
		carry = fn(carry, v)
	}
	return carry
}

// Sum sums numeric values extracted by fn.
func Sum[T any](s []T, fn func(T) float64) float64 {
	return Reduce(s, 0.0, func(acc float64, v T) float64 { return acc + fn(v) })
}

// Flatten merges a slice-of-slices into a single slice.
func Flatten[T any](s [][]T) []T {
	var out []T
	for _, inner := range s {
		out = append(out, inner...)
	}
	return out
}

// Reverse returns a new slice with elements in reverse order.
func Reverse[T any](s []T) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

// Take returns the first n elements.
func Take[T any](s []T, n int) []T {
	if n >= len(s) {
		return s
	}
	return s[:n]
}

// Skip returns s without the first n elements.
func Skip[T any](s []T, n int) []T {
	if n >= len(s) {
		return nil
	}
	return s[n:]
}

// KeyBy turns s into a map using the key produced by fn.
// If two elements produce the same key, the last one wins.
func KeyBy[T any, K comparable](s []T, fn func(T) K) map[K]T {
	out := make(map[K]T, len(s))
	for _, v := range s {
		out[fn(v)] = v
	}
	return out
}

// Paginate returns one page from s (1-indexed page, size items per page).
func Paginate[T any](s []T, page, size int) []T {
	if page < 1 {
		page = 1
	}
	start := (page - 1) * size
	if start >= len(s) {
		return nil
	}
	end := start + size
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
