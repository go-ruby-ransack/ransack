// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import "testing"

func TestToFloat(t *testing.T) {
	cases := []struct {
		v    any
		want float64
		ok   bool
	}{
		{int(3), 3, true},
		{int64(-4), -4, true},
		{uint(5), 5, true},
		{float32(1.5), 1.5, true},
		{float64(2.25), 2.25, true},
		{"  18  ", 18, true},
		{"notnum", 0, false},
		{true, 0, false}, // bool falls through default
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := toFloat(c.v)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("toFloat(%#v) = %v,%v want %v,%v", c.v, got, ok, c.want, c.ok)
		}
	}
}

func TestToString(t *testing.T) {
	if toString(nil) != "" {
		t.Fatal("nil should stringify to empty")
	}
	if toString(42) != "42" {
		t.Fatal("42")
	}
}

func TestCompareAndEqual(t *testing.T) {
	if compareGeneric(1, 2) != -1 || compareGeneric(2, 1) != 1 || compareGeneric(2, 2) != 0 {
		t.Fatal("numeric compare")
	}
	if compareGeneric("a", "b") != -1 || compareGeneric("b", "a") != 1 || compareGeneric("a", "a") != 0 {
		t.Fatal("string compare")
	}
	if !equalValues("7", 7) {
		t.Fatal("numeric equal across types")
	}
	if !equalValues("x", "x") || equalValues("x", "y") {
		t.Fatal("string equal")
	}
}

func TestBlankPresentNil(t *testing.T) {
	if !isBlankVal(nil) || !isBlankVal("  ") || !isBlankVal([]any{}) {
		t.Fatal("blank cases")
	}
	if isBlankVal("x") || isBlankVal(0) {
		t.Fatal("non-blank cases")
	}
	if isPresent(nil) || !isPresent("x") {
		t.Fatal("present")
	}
	if !isNil(nil) || isNil(1) {
		t.Fatal("nil")
	}
}

func TestParseBool(t *testing.T) {
	truthy := []any{true, "1", "true", "T", "yes", "on", 1, 2.0}
	for _, v := range truthy {
		if !parseBool(v) {
			t.Errorf("parseBool(%#v) want true", v)
		}
	}
	falsy := []any{false, "0", "false", "no", "", nil, 0, []any{"x"}}
	for _, v := range falsy {
		if parseBool(v) {
			t.Errorf("parseBool(%#v) want false", v)
		}
	}
}

func TestToSliceAndFilterBlanks(t *testing.T) {
	if got := toSlice([]any{1, 2}); len(got) != 2 {
		t.Fatal("[]any")
	}
	if got := toSlice([]string{"a", "b", "c"}); len(got) != 3 {
		t.Fatal("[]string")
	}
	if got := toSlice("solo"); len(got) != 1 || got[0] != "solo" {
		t.Fatal("scalar wrap")
	}
	if got := filterBlanks([]any{"a", "", "  ", "b", nil}); len(got) != 2 {
		t.Fatalf("filterBlanks kept %v", got)
	}
}

func TestLikeMatch(t *testing.T) {
	if !likeMatch("hello", "he%o") {
		t.Fatal("% wildcard")
	}
	if !likeMatch("cat", "c_t") {
		t.Fatal("_ wildcard")
	}
	if likeMatch("hello", "world") {
		t.Fatal("no match")
	}
	if !likeMatch("a.b", "a.b") {
		t.Fatal("literal dot escaped")
	}
}
