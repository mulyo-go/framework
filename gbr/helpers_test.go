package gbr

import (
	"html/template"
	"testing"
)

func TestBuiltinHelpers(t *testing.T) {
	// String helpers
	if ucfirst("hello") != "Hello" {
		t.Errorf("ucfirst failed")
	}
	if toKebab("Hello World") != "hello-world" {
		t.Errorf("toKebab failed")
	}
	if toSnake("Hello World") != "hello_world" {
		t.Errorf("toSnake failed")
	}
	if toCamel("hello world") != "helloWorld" {
		t.Errorf("toCamel failed")
	}
	if toStudly("hello world") != "HelloWorld" {
		t.Errorf("toStudly failed")
	}
	if toSlug("Hello World! 123") != "hello-world-123" {
		t.Errorf("toSlug failed: got %s", toSlug("Hello World! 123"))
	}
	if strLimit("Hello World", 5, "...") != "Hello..." {
		t.Errorf("strLimit failed")
	}
	if strReverse("abc") != "cba" {
		t.Errorf("strReverse failed")
	}
	if wordCount("hello world from go") != 4 {
		t.Errorf("wordCount failed")
	}
	if strSubstr("abcdef", 1, 3) != "bcd" {
		t.Errorf("strSubstr failed")
	}
	if squish("  hello   world  ") != "hello world" {
		t.Errorf("squish failed")
	}
	if headline("user_profile_id") != "User Profile Id" {
		t.Errorf("headline failed")
	}
	if strPlural("user") != "users" || strPlural("category") != "categories" {
		t.Errorf("strPlural failed")
	}
	if strSingular("users") != "user" || strSingular("categories") != "category" {
		t.Errorf("strSingular failed")
	}
	if padLeft("5", 3, "0") != "005" {
		t.Errorf("padLeft failed")
	}
	if padRight("5", 3, "0") != "500" {
		t.Errorf("padRight failed")
	}

	// Number helpers
	if numAdd(1, 2, 3) != 6 {
		t.Errorf("numAdd failed")
	}
	if numSub(10, 3) != 7 {
		t.Errorf("numSub failed")
	}
	if numMul(4, 5) != 20 {
		t.Errorf("numMul failed")
	}
	if numDiv(10, 2) != 5 {
		t.Errorf("numDiv failed")
	}
	if numMod(10, 3) != 1 {
		t.Errorf("numMod failed")
	}
	if numMin(5, 3) != 3 {
		t.Errorf("numMin failed")
	}
	if numMax(5, 3) != 5 {
		t.Errorf("numMax failed")
	}
	if formatNumber(1234567.89, 2, ",", ".") != "1.234.567,89" {
		t.Errorf("formatNumber failed: got %s", formatNumber(1234567.89, 2, ",", "."))
	}
	if formatPercent(0.755, 1) != "75.5%" {
		t.Errorf("formatPercent failed: got %s", formatPercent(0.755, 1))
	}
	if formatBytes(1048576) != "1.00 MB" {
		t.Errorf("formatBytes failed: got %s", formatBytes(1048576))
	}
	if formatIDR(1500000) != "Rp 1.500.000" {
		t.Errorf("formatIDR failed: got %s", formatIDR(1500000))
	}

	// Logic & Array
	if ternary(true, "yes", "no") != "yes" {
		t.Errorf("ternary failed")
	}
	if !isEmpty("") || !isEmpty(0) || !isEmpty(nil) || isEmpty("abc") {
		t.Errorf("isEmpty failed")
	}
	if !eqAny("b", "a", "b", "c") {
		t.Errorf("eqAny failed")
	}
	if !inArray("admin", []string{"user", "admin"}) {
		t.Errorf("inArray failed")
	}
	m := map[string]int{"a": 1, "b": 2}
	if !hasKey(m, "a") {
		t.Errorf("hasKey failed")
	}
	d, err := dictHelper("key1", "val1", "key2", "val2")
	if err != nil || d["key1"] != "val1" {
		t.Errorf("dictHelper failed")
	}
	l := listHelper("a", "b", "c")
	if len(l) != 3 {
		t.Errorf("listHelper failed")
	}

	// String utils
	if nl2br("a\nb") != template.HTML("a<br/>b") {
		t.Errorf("nl2br failed: got %s", nl2br("a\nb"))
	}
	if excerpt("Halo nama saya adalah Antigravity", 15) != "Halo nama saya..." {
		t.Errorf("excerpt failed: got %s", excerpt("Halo nama saya adalah Antigravity", 15))
	}
	if maskString("08123456789", 4, 4, "*") != "0812****789" {
		t.Errorf("maskString failed: got %s", maskString("08123456789", 4, 4, "*"))
	}
}

func TestResolveViewPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sample/View/Blade/Loop", "Module/Sample/View/Blade/Loop.gbr.html"},
		{"Sample/View/Blade/Loop.gbr.html", "Module/Sample/View/Blade/Loop.gbr.html"},
		{"Template/Default/layout.gbr.html", "Template/Default/layout.gbr.html"},
		{"Metronic/layout.gbr.html", "Template/Metronic/layout.gbr.html"},
		{"Metronic/layout", "Template/Metronic/layout.gbr.html"},
	}

	for _, tt := range tests {
		res := resolveViewPath(tt.input)
		if res != tt.expected {
			t.Errorf("resolveViewPath(%q) = %q; want %q", tt.input, res, tt.expected)
		}
	}
}
