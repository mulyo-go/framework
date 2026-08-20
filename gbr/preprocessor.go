package gbr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var gbrSyntaxPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\{\{--`),
	regexp.MustCompile(`\{!!`),
	regexp.MustCompile(`@\{\{`),
	regexp.MustCompile(`@(?:if|elseif|unless|isset|empty|switch|case|foreach|forelse|for|while|extends|section|yield|include|includeIf|includeWhen|includeUnless|includeFirst|props|aware|push|prepend|pushOnce|stack|method|error|checked|selected|disabled|readonly|required|session|env|inject|json|js|dump|dd|class|style|fragment)\s*\(`),
	regexp.MustCompile(`@(?:else|endif|endunless|endisset|endempty|default|endswitch|endforeach|endforelse|endfor|endwhile|show|parent|endsection|endpush|endprepend|endPushOnce|csrf|auth(?:\([^)]*\))?|endauth|guest|endguest|endsession|production|endproduction|php|endphp|verbatim|endverbatim|once|endonce|endfragment)\b`),
}

func hasGbrSyntax(content string) bool {
	for _, pattern := range gbrSyntaxPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// preprocessGbr converts GBR / Blade @directives to Go template syntax
func preprocessGbr(content string) string {
	// Skip preprocessing if no directives found
	if !hasGbrSyntax(content) {
		return content
	}

	// 1. Remove comments: {{-- ... --}}
	content = regexp.MustCompile(`\{\{--.*?--\}\}`).ReplaceAllString(content, "")

	// 2. Escape: @{{ ... }} → literal {{ ... }}
	content = handleEscaped(content)

	// 3. Raw HTML: {!! $var !!} → {{ safeHTML .Var }}
	content = regexp.MustCompile(`\{!!\s*\$(\w+)\s*!!\}`).ReplaceAllString(content, `{{ safeHTML .$1 }}`)

	// 4. Verbatim blocks: @verbatim ... @endverbatim → escape {{ }}
	content = handleVerbatim(content)

	// 5. Once blocks
	content = handleOnce(content)

	// 6. Stack directives: @push, @prepend, @stack, @pushOnce
	content = handleStacks(content)

	// 7. Layout directives: @extends, @section, @yield, @show, @parent
	content = handleLayout(content)

	// 8. Include directives: @include, @includeIf, @includeWhen, @includeUnless, @includeFirst
	content = handleIncludes(content)

	// 9. Component directives: <x-*>, @props, @aware, @slot
	content = handleComponents(content)

	// 10. Form helpers: @csrf, @method, @error, @checked, @selected, @disabled, @readonly, @required
	content = handleFormHelpers(content)

	// 11. Auth directives: @auth, @guest, @endauth, @endguest
	content = handleAuth(content)

	// 12. Session directive: @session, @endsession
	content = handleSession(content)

	// 13. Environment: @env, @endenv, @production, @endproduction
	content = handleEnvironment(content)

	// 14. Inject: @inject('var', 'Service')
	content = handleInject(content)

	// 15. JSON/JS: @json, @js
	content = handleJsonJs(content)

	// 16. Debug: @dump, @dd
	content = handleDebug(content)

	// 17. Attribute helpers: @class, @style
	content = handleAttributeHelpers(content)

	// 18. Fragment: @fragment, @endfragment
	content = handleFragment(content)

	// 19. Switch/Case
	content = handleSwitch(content)

	// 20. Conditionals: @if, @elseif, @else, @endif, @unless, @isset, @empty
	content = handleConditionals(content)

	// 21. Loops: @foreach, @forelse, @for, @while, @break, @continue
	content = handleLoops(content)

	// 22. PHP blocks (strip or ignore)
	content = regexp.MustCompile(`(?s)@php.*?@endphp`).ReplaceAllString(content, "")

	return content
}

func handleEscaped(content string) string {
	re := regexp.MustCompile(`@\{\{([\s\S]*?)\}\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		inner := parts[1]
		return `{{ "{{" }}` + inner + `{{ "}}" }}`
	})
}

// convertVar converts $var to .Var, $user->name to .User.Name
func convertVar(expr string) string {
	// $user->name → .User.Name
	expr = regexp.MustCompile(`\$(\w+)->(\w+)`).ReplaceAllString(expr, `.${capitalize($1)}.${capitalize($2)}`)

	// $user['name'] → index .User "name"
	expr = regexp.MustCompile(`\$(\w+)\['(\w+)'\]`).ReplaceAllStringFunc(expr, func(match string) string {
		parts := regexp.MustCompile(`\$(\w+)\['(\w+)'\]`).FindStringSubmatch(match)
		if len(parts) >= 3 {
			return fmt.Sprintf(`index .%s "%s"`, capitalize(parts[1]), parts[2])
		}
		return match
	})

	// $var → .Var (only if followed by space, ), or end of expression)
	expr = regexp.MustCompile(`\$(\w+)`).ReplaceAllStringFunc(expr, func(match string) string {
		parts := regexp.MustCompile(`\$(\w+)`).FindStringSubmatch(match)
		if len(parts) >= 2 {
			return "." + capitalize(parts[1])
		}
		return match
	})

	return expr
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// --- Verbatim ---
func handleVerbatim(content string) string {
	re := regexp.MustCompile(`(?s)@verbatim(.*?)@endverbatim`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		inner := re.FindStringSubmatch(match)[1]
		// Gunakan placeholder agar replacement {{ dan }} tidak saling menimpa.
		inner = strings.ReplaceAll(inner, "{{", "__GBR_VERBATIM_OPEN__")
		inner = strings.ReplaceAll(inner, "}}", "__GBR_VERBATIM_CLOSE__")
		inner = strings.ReplaceAll(inner, "__GBR_VERBATIM_OPEN__", `{{ "{{" }}`)
		inner = strings.ReplaceAll(inner, "__GBR_VERBATIM_CLOSE__", `{{ "}}" }}`)
		return inner
	})
}

// --- Once ---
var onceBlocks = map[string]bool{}

func handleOnce(content string) string {
	re := regexp.MustCompile(`(?s)@once(.*?)@endonce`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		inner := re.FindStringSubmatch(match)[1]
		hash := fmt.Sprintf("_once_%d", len(onceBlocks))
		onceBlocks[hash] = true
		return fmt.Sprintf(`{{ if not (index . "__once_%s") }}{{ $__once_%s := true }}%s{{ end }}`, hash, hash, inner)
	})
}

// --- Stacks ---
func handleStacks(content string) string {
	// @push('name') ... @endpush → store in __stack_name
	re := regexp.MustCompile(`(?s)@push\('(\w+)'\)(.*?)@endpush`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		name := parts[1]
		inner := strings.TrimSpace(parts[2])
		return fmt.Sprintf(`{{ .__stacks.Set "%s" %s }}`, name, strconv.Quote(inner))
	})

	// @prepend('name') ... @endprepend
	re = regexp.MustCompile(`(?s)@prepend\('(\w+)'\)(.*?)@endprepend`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		name := parts[1]
		inner := strings.TrimSpace(parts[2])
		return fmt.Sprintf(`{{ .__stacks.Prepend "%s" %s }}`, name, strconv.Quote(inner))
	})

	// @pushOnce('name') ... @endPushOnce
	re = regexp.MustCompile(`(?is)@pushOnce\('(\w+)'\)(.*?)@endPushOnce`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		name := parts[1]
		inner := strings.TrimSpace(parts[2])
		return fmt.Sprintf(`{{ if not (index . "__pushOnce_%s") }}{{ .__stacks.Set "%s" %s }}{{ end }}`, name, name, strconv.Quote(inner))
	})

	// @stack('name') → render stack
	content = regexp.MustCompile(`@stack\('(\w+)'\)`).ReplaceAllString(content, `{{ .__stacks.Render "$1" }}`)

	return content
}

// --- Layout ---
func handleLayout(content string) string {
	// @extends('layout') → {{ template "layout" . }}
	content = regexp.MustCompile(`@extends\('([^']+)'\)`).ReplaceAllString(content, `{{ template "$1" . }}`)

	// @section('name') → {{ define "name" }}
	content = regexp.MustCompile(`@section\('([^']+)'\)`).ReplaceAllString(content, `{{ define "$1" }}`)

	// @endsection → {{ end }}
	content = strings.ReplaceAll(content, `@endsection`, `{{ end }}`)

	// @yield('name') → {{ template "name" . }}
	content = regexp.MustCompile(`@yield\('([^']+)'\)`).ReplaceAllString(content, `{{ template "$1" . }}`)

	// @show → {{ end }} (section with default that can be overridden)
	content = strings.ReplaceAll(content, `@show`, `{{ end }}`)

	// @parent → (placeholder for parent content)
	content = strings.ReplaceAll(content, `@parent`, `{{ "<!-- parent -->" }}`)

	// @fragment('name') ... @endfragment → {{ define "name" }} ... {{ end }}
	content = regexp.MustCompile(`(?s)@fragment\('([^']+)'\)(.*?)@endfragment`).ReplaceAllString(content, `{{ define "$1" }}$2{{ end }}`)

	return content
}

// --- Includes ---
func handleIncludes(content string) string {
	// @include('partial') or @include('partial', $data) → {{ template "partial" . }}
	content = regexp.MustCompile(`@include\('([^']+)'(?:,\s*[^)]+)?\)`).ReplaceAllString(content, `{{ template "$1" . }}`)

	// @includeIf('partial') or @includeIf('partial', $data) → {{ template "partial" . }}
	content = regexp.MustCompile(`@includeIf\('([^']+)'(?:,\s*[^)]+)?\)`).ReplaceAllString(content, `{{ template "$1" . }}`)

	// @includeWhen($condition, 'partial') → {{ if .Condition }}{{ template "partial" . }}{{ end }}
	content = regexp.MustCompile(`@includeWhen\(\$(\w+),\s*'([^']+)'\)`).ReplaceAllString(content, `{{ if .$1 }}{{ template "$2" . }}{{ end }}`)

	// @includeUnless($condition, 'partial') → {{ if not .Condition }}{{ template "partial" . }}{{ end }}
	content = regexp.MustCompile(`@includeUnless\(\$(\w+),\s*'([^']+)'\)`).ReplaceAllString(content, `{{ if not .$1 }}{{ template "$2" . }}{{ end }}`)

	// @includeFirst(['a', 'b']) → try a first, then b
	content = regexp.MustCompile(`@includeFirst\(\[([^\]]+)\]\)`).ReplaceAllStringFunc(content, func(match string) string {
		parts := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(match, -1)
		if len(parts) == 0 {
			return ""
		}
		return fmt.Sprintf(`{{ template "%s" . }}`, parts[0][1])
	})

	return content
}

// --- Components ---
func handleComponents(content string) string {
	// <x-component-name /> or <x-component-name>
	content = regexp.MustCompile(`<x-(\w+)\s*/>`).ReplaceAllString(content, `{{ template "components/$1" . }}`)

	// <x-component-name>content</x-component-name>
	content = regexp.MustCompile(`(?s)<x-(\w+)>(.*?)</x-\w+>`).ReplaceAllStringFunc(content, func(match string) string {
		parts := regexp.MustCompile(`(?s)<x-(\w+)>(.*?)</x-\w+>`).FindStringSubmatch(match)
		if len(parts) >= 3 {
			return parts[2]
		}
		return match
	})

	// @props([...])
	content = regexp.MustCompile(`@props\(\[.*?\]\)`).ReplaceAllString(content, ``)

	// @aware([...])
	content = regexp.MustCompile(`@aware\(\[.*?\]\)`).ReplaceAllString(content, ``)

	// <x-slot:name>content</x-slot:name> → {{ define "slot_name" }}content{{ end }}
	content = regexp.MustCompile(`(?s)<x-slot:(\w+)>(.*?)</x-slot:\w+>`).ReplaceAllStringFunc(content, func(match string) string {
		parts := regexp.MustCompile(`(?s)<x-slot:(\w+)>(.*?)</x-slot:\w+>`).FindStringSubmatch(match)
		if len(parts) >= 3 {
			return fmt.Sprintf(`{{ define "slot_%s" }}%s{{ end }}`, parts[1], parts[2])
		}
		return match
	})

	return content
}

// --- Form Helpers ---
func handleFormHelpers(content string) string {
	// @csrf → hidden input with CSRF token
	content = strings.ReplaceAll(content, `@csrf`, `<input type="hidden" name="_token" value="{{ .CsrfToken }}">`)

	// @method('PUT') → hidden input for method spoofing
	content = regexp.MustCompile(`@method\('([^']+)'\)`).ReplaceAllString(content, `<input type="hidden" name="_method" value="$1">`)

	// @error('field') ... @enderror → {{ if .Errors.field }} ... {{ end }}
	content = regexp.MustCompile(`@error\('([^']+)'\)`).ReplaceAllString(content, `{{ with index .Errors "$1" }}`)
	content = strings.ReplaceAll(content, `@enderror`, `{{ end }}`)

	// @checked($var) → {{ if $var }}checked{{ end }}
	content = regexp.MustCompile(`@checked\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@checked\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}checked{{ end }}`, capitalize(varName))
	})

	// @selected($var) → {{ if $var }}selected{{ end }}
	content = regexp.MustCompile(`@selected\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@selected\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}selected{{ end }}`, capitalize(varName))
	})

	// @disabled($var) → {{ if $var }}disabled{{ end }}
	content = regexp.MustCompile(`@disabled\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@disabled\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}disabled{{ end }}`, capitalize(varName))
	})

	// @readonly($var) → {{ if $var }}readonly{{ end }}
	content = regexp.MustCompile(`@readonly\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@readonly\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}readonly{{ end }}`, capitalize(varName))
	})

	// @required($var) → {{ if $var }}required{{ end }}
	content = regexp.MustCompile(`@required\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@required\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}required{{ end }}`, capitalize(varName))
	})

	return content
}

// --- Auth ---
func handleAuth(content string) string {
	if strings.Contains(content, `@endauth`) {
		content = regexp.MustCompile(`@auth(?:\([^)]*\))?`).ReplaceAllString(content, `{{ if .LoggedIn }}`)
		content = strings.ReplaceAll(content, `@endauth`, `{{ end }}`)
	}

	if strings.Contains(content, `@endguest`) {
		content = regexp.MustCompile(`@guest(?:\([^)]*\))?`).ReplaceAllString(content, `{{ if not .LoggedIn }}`)
		content = strings.ReplaceAll(content, `@endguest`, `{{ end }}`)
	}

	return content
}

// --- Session ---
func handleSession(content string) string {
	// @session('status') ... @endsession → {{ with .FlashStatus }} ... {{ end }}
	content = regexp.MustCompile(`@session\('([^']+)'\)`).ReplaceAllStringFunc(content, func(match string) string {
		key := regexp.MustCompile(`@session\('([^']+)'\)`).FindStringSubmatch(match)[1]
		return fmt.Sprintf(`{{ with index .Flashes "%s" }}`, key)
	})
	content = strings.ReplaceAll(content, `@endsession`, `{{ end }}`)

	return content
}

// --- Environment ---
func handleEnvironment(content string) string {
	// @production ... @endproduction
	content = strings.ReplaceAll(content, `@production`, `{{ if eq .Env "production" }}`)
	content = strings.ReplaceAll(content, `@endproduction`, `{{ end }}`)

	// @env('local', 'staging') ... @endenv
	content = regexp.MustCompile(`@env\(([^)]+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		envs := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(match, -1)
		var conds []string
		for _, e := range envs {
			conds = append(conds, fmt.Sprintf(`eq .Env "%s"`, e[1]))
		}
		if len(conds) == 1 {
			return fmt.Sprintf(`{{ if %s }}`, conds[0])
		}
		return fmt.Sprintf(`{{ if or %s }}`, strings.Join(conds, " "))
	})
	content = strings.ReplaceAll(content, `@endenv`, `{{ end }}`)

	return content
}

// --- Inject ---
func handleInject(content string) string {
	// @inject('metrics', 'App\Services\MetricsService') → (comment/ignored in Go template context)
	content = regexp.MustCompile(`@inject\('([^']+)',\s*'([^']+)'\)`).ReplaceAllString(content, ``)
	return content
}

// --- JSON / JS ---
func handleJsonJs(content string) string {
	// @json($data) → {{ json .Data }}
	content = regexp.MustCompile(`@json\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@json\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ json .%s }}`, capitalize(varName))
	})

	// @js($data) → {{ js .Data }}
	content = regexp.MustCompile(`@js\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@js\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ js .%s }}`, capitalize(varName))
	})

	return content
}

// --- Debug ---
func handleDebug(content string) string {
	// @dump($var) → {{ dump .Var }}
	content = regexp.MustCompile(`@dump\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@dump\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ dump .%s }}`, capitalize(varName))
	})

	// @dd($var) → {{ dd .Var }}
	content = regexp.MustCompile(`@dd\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@dd\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ dd .%s }}`, capitalize(varName))
	})

	return content
}

// --- Attribute Helpers ---
func handleAttributeHelpers(content string) string {
	// @class(['p-4', 'bg-red' => $hasError])
	// Sederhana: ubah ke helper classNames
	content = regexp.MustCompile(`@class\(\[([^\]]+)\]\)`).ReplaceAllStringFunc(content, func(match string) string {
		inner := regexp.MustCompile(`@class\(\[([^\]]+)\]\)`).FindStringSubmatch(match)[1]
		// Parse string literals
		classes := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(inner, -1)
		var classList []string
		for _, c := range classes {
			classList = append(classList, c[1])
		}
		return fmt.Sprintf(`class="%s"`, strings.Join(classList, " "))
	})

	// @style(['background-color: red', 'color: white' => $isActive])
	content = regexp.MustCompile(`@style\(\[([^\]]+)\]\)`).ReplaceAllStringFunc(content, func(match string) string {
		inner := regexp.MustCompile(`@style\(\[([^\]]+)\]\)`).FindStringSubmatch(match)[1]
		styles := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(inner, -1)
		var styleList []string
		for _, s := range styles {
			styleList = append(styleList, s[1])
		}
		return fmt.Sprintf(`style="%s"`, strings.Join(styleList, "; "))
	})

	return content
}

// --- Fragment ---
func handleFragment(content string) string {
	// Handled in handleLayout: @fragment('name') ... @endfragment
	return content
}

// --- Switch / Case ---
func handleSwitch(content string) string {
	// @switch($var) ... @case(val) ... @break ... @default ... @endswitch
	// Convert to if/elseif chain:
	// @switch($type)
	//   @case(1) ... @break
	//   @case(2) ... @break
	//   @default ...
	// @endswitch
	// Catatan: Go template tidak punya switch, jadi ubah ke if/else
	content = regexp.MustCompile(`@switch\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@switch\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ $__switch := .%s }}`, capitalize(varName))
	})

	// First @case after switch → {{ if eq $__switch val }}
	// Subsequent @case → {{ else if eq $__switch val }}
	isFirstCase := true
	reCase := regexp.MustCompile(`@case\(([^)]+)\)`)
	content = reCase.ReplaceAllStringFunc(content, func(match string) string {
		val := reCase.FindStringSubmatch(match)[1]
		if isFirstCase {
			isFirstCase = false
			return fmt.Sprintf(`{{ if eq $__switch %s }}`, val)
		}
		return fmt.Sprintf(`{{ else if eq $__switch %s }}`, val)
	})

	content = strings.ReplaceAll(content, `@default`, `{{ else }}`)
	content = strings.ReplaceAll(content, `@endswitch`, `{{ end }}`)
	content = strings.ReplaceAll(content, `@break`, "")

	return content
}

// --- Conditionals ---
func handleConditionals(content string) string {
	// @if($var) or @if($a == $b)
	content = regexp.MustCompile(`@if\(([^)]+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		expr := regexp.MustCompile(`@if\(([^)]+)\)`).FindStringSubmatch(match)[1]
		return fmt.Sprintf(`{{ if %s }}`, convertCondition(expr))
	})

	// @elseif($var)
	content = regexp.MustCompile(`@elseif\(([^)]+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		expr := regexp.MustCompile(`@elseif\(([^)]+)\)`).FindStringSubmatch(match)[1]
		return fmt.Sprintf(`{{ else if %s }}`, convertCondition(expr))
	})

	// @else
	content = strings.ReplaceAll(content, `@else`, `{{ else }}`)

	// @endif
	content = strings.ReplaceAll(content, `@endif`, `{{ end }}`)

	// @unless($var) → {{ if not $var }}
	content = regexp.MustCompile(`@unless\(([^)]+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		expr := regexp.MustCompile(`@unless\(([^)]+)\)`).FindStringSubmatch(match)[1]
		return fmt.Sprintf(`{{ if not (%s) }}`, convertCondition(expr))
	})
	content = strings.ReplaceAll(content, `@endunless`, `{{ end }}`)

	// @isset($var) → {{ if $var }}
	content = regexp.MustCompile(`@isset\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@isset\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if .%s }}`, capitalize(varName))
	})
	content = strings.ReplaceAll(content, `@endisset`, `{{ end }}`)

	// @empty($var) → {{ if empty .Var }}
	content = regexp.MustCompile(`@empty\((\$?\w+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		varName := regexp.MustCompile(`@empty\((\$?\w+)\)`).FindStringSubmatch(match)[1]
		varName = strings.TrimPrefix(varName, "$")
		return fmt.Sprintf(`{{ if empty .%s }}`, capitalize(varName))
	})
	content = strings.ReplaceAll(content, `@endempty`, `{{ end }}`)

	return content
}

// convertCondition converts PHP-style condition to Go template condition
func convertCondition(expr string) string {
	expr = strings.TrimSpace(expr)

	// Handle == comparison: $a == $b → eq $a $b
	reEq := regexp.MustCompile(`(\$[\w->]+|\w+)\s*==\s*(\$[\w->]+|'[^']*'|"[^"]*"|\d+)`)
	if reEq.MatchString(expr) {
		parts := reEq.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`eq %s %s`, left, right)
	}

	// Handle != comparison: $a != $b → ne $a $b
	reNe := regexp.MustCompile(`(\$[\w->]+|\w+)\s*!=\s*(\$[\w->]+|'[^']*'|"[^"]*"|\d+)`)
	if reNe.MatchString(expr) {
		parts := reNe.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`ne %s %s`, left, right)
	}

	// Handle > comparison: $a > $b → gt $a $b
	reGt := regexp.MustCompile(`(\$[\w->]+|\w+)\s*>\s*(\$[\w->]+|\d+)`)
	if reGt.MatchString(expr) {
		parts := reGt.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`gt %s %s`, left, right)
	}

	// Handle < comparison: $a < $b → lt $a $b
	reLt := regexp.MustCompile(`(\$[\w->]+|\w+)\s*<\s*(\$[\w->]+|\d+)`)
	if reLt.MatchString(expr) {
		parts := reLt.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`lt %s %s`, left, right)
	}

	// Handle >= comparison: $a >= $b → ge $a $b
	reGe := regexp.MustCompile(`(\$[\w->]+|\w+)\s*>=\s*(\$[\w->]+|\d+)`)
	if reGe.MatchString(expr) {
		parts := reGe.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`ge %s %s`, left, right)
	}

	// Handle <= comparison: $a <= $b → le $a $b
	reLe := regexp.MustCompile(`(\$[\w->]+|\w+)\s*<=\s*(\$[\w->]+|\d+)`)
	if reLe.MatchString(expr) {
		parts := reLe.FindStringSubmatch(expr)
		left := convertVar(parts[1])
		right := convertVar(parts[2])
		return fmt.Sprintf(`le %s %s`, left, right)
	}

	// Simple variable: $var → .Var
	return convertVar(expr)
}

// --- Loops ---
func handleLoops(content string) string {
	// @foreach($items as $item) → {{ range $item := .Items }}
	reForeach := regexp.MustCompile(`@foreach\(\$(\w+)\s+as\s+\$(\w+)\)`)
	content = reForeach.ReplaceAllStringFunc(content, func(match string) string {
		parts := reForeach.FindStringSubmatch(match)
		items := capitalize(parts[1])
		item := parts[2]
		return fmt.Sprintf(`{{ range $%s := .%s }}`, item, items)
	})

	// @foreach($items as $key => $value) → {{ range $key, $value := .Items }}
	reForeachKV := regexp.MustCompile(`@foreach\(\$(\w+)\s+as\s+\$(\w+)\s*=>\s*\$(\w+)\)`)
	content = reForeachKV.ReplaceAllStringFunc(content, func(match string) string {
		parts := reForeachKV.FindStringSubmatch(match)
		items := capitalize(parts[1])
		key := parts[2]
		val := parts[3]
		return fmt.Sprintf(`{{ range $%s, $%s := .%s }}`, key, val, items)
	})

	content = strings.ReplaceAll(content, `@endforeach`, `{{ end }}`)

	// @forelse($items as $item) ... @empty ... @endforelse
	reForelse := regexp.MustCompile(`@forelse\(\$(\w+)\s+as\s+\$(\w+)\)`)
	content = reForelse.ReplaceAllStringFunc(content, func(match string) string {
		parts := reForelse.FindStringSubmatch(match)
		items := capitalize(parts[1])
		item := parts[2]
		return fmt.Sprintf(`{{ range $%s := .%s }}`, item, items)
	})
	// In forelse, @empty translates to {{ else }}
	// Handled by @empty inside forelse:
	content = strings.ReplaceAll(content, `@endforelse`, `{{ end }}`)

	// @for($i = 0; $i < 10; $i++) → (not natively supported in Go template, warn/convert to range)
	// Sederhana: @for($i = 0; $i < $n; $i++) → {{ range $i := seq 0 $n }}
	reFor := regexp.MustCompile(`@for\(\$(\w+)\s*=\s*(\d+);\s*\$\w+\s*<\s*(\d+|\$\w+);\s*\$\w+\+\+\)`)
	content = reFor.ReplaceAllStringFunc(content, func(match string) string {
		parts := reFor.FindStringSubmatch(match)
		varName := parts[1]
		start := parts[2]
		end := parts[3]
		if strings.HasPrefix(end, "$") {
			end = "." + capitalize(end[1:])
		}
		return fmt.Sprintf(`{{ range $%s := seq %s %s }}`, varName, start, end)
	})
	content = strings.ReplaceAll(content, `@endfor`, `{{ end }}`)

	// @while($condition) → {{ /* @while not supported in Go templates */ }}
	content = regexp.MustCompile(`@while\([^)]+\)`).ReplaceAllString(content, `{{ /* @while */ }}`)
	content = strings.ReplaceAll(content, `@endwhile`, `{{ /* @endwhile */ }}`)

	// @continue
	content = strings.ReplaceAll(content, `@continue`, `{{ continue }}`)

	return content
}
