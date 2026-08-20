package gbr

import (
	"strings"
	"testing"
)

func TestPreprocessorBasics(t *testing.T) {
	// Test escaping @{{ }}
	input := `Hello @{{ name }}!`
	expected := `Hello {{ "{{" }} name {{ "}}" }}!`
	res := preprocessGbr(input)
	if res != expected {
		t.Errorf("preprocess @{{ }} failed: got %q, want %q", res, expected)
	}

	// Test @if conditional
	input = `@if($isAdmin)<span>Admin</span>@endif`
	res = preprocessGbr(input)
	if !strings.Contains(res, `{{ if .IsAdmin }}`) || !strings.Contains(res, `{{ end }}`) {
		t.Errorf("preprocess @if failed: got %q", res)
	}

	// Test @foreach
	input = `@foreach($users as $user)<div>{{ $user.name }}</div>@endforeach`
	res = preprocessGbr(input)
	if !strings.Contains(res, `{{ range $user := .Users }}`) || !strings.Contains(res, `{{ end }}`) {
		t.Errorf("preprocess @foreach failed: got %q", res)
	}

	// Test @csrf
	input = `<form>@csrf</form>`
	res = preprocessGbr(input)
	if !strings.Contains(res, `<input type="hidden" name="_token" value="{{ .CsrfToken }}">`) {
		t.Errorf("preprocess @csrf failed: got %q", res)
	}
}

func TestStackUnescapesTabAndNewlines(t *testing.T) {
	stack := &templateStack{items: map[string]string{}}
	input := `\tvar $modal = $('#menuModal');\n\tvar regex = /^\d+$/;`
	stack.Set("js", input)
	res := string(stack.Render("js"))
	if strings.Contains(res, `\t`) {
		t.Errorf("rendered stack output should not contain literal \\t string, got:\n%s", res)
	}
	if !strings.Contains(res, "\tvar $modal") {
		t.Errorf("rendered stack output should contain actual tab byte, got:\n%s", res)
	}
}
