// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type segment struct {
	key     string
	index   int
	isIndex bool
}

type woPath []segment

var bareKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// parseWOPath accepts a bare or Go-quoted first key, then .key, [N] or ["key"] segments.
func parseWOPath(s string) (woPath, error) {
	fail := func(reason string) (woPath, error) {
		return nil, fmt.Errorf("invalid manifest_wo_version key %q: %s", s, reason)
	}
	if s == "" {
		return fail("empty path")
	}

	var p woPath

	i := 0

	if s[0] == '[' {
		if len(s) < 2 || s[1] != '"' {
			return fail("the first segment must be a key")
		}

		key, next, err := parseQuotedSegment(s, i)
		if err != nil {
			return fail(err.Error())
		}

		p = append(p, segment{key: key})
		i = next
	} else {
		end := strings.IndexAny(s, ".[")
		if end < 0 {
			end = len(s)
		}

		key := s[:end]
		if key == "" {
			return fail("empty key")
		}

		if strings.ContainsAny(key, "]\"") {
			return fail("invalid character in key")
		}

		p = append(p, segment{key: key})
		i = end
	}

	for i < len(s) {
		switch s[i] {
		case '.':
			start := i + 1

			end := start
			for end < len(s) && s[end] != '.' && s[end] != '[' {
				end++
			}

			key := s[start:end]
			if key == "" {
				return fail("empty key")
			}

			if strings.ContainsAny(key, "]\"") {
				return fail("invalid character in key")
			}

			p = append(p, segment{key: key})
			i = end
		case '[':
			if i+1 < len(s) && s[i+1] == '"' {
				key, next, err := parseQuotedSegment(s, i)
				if err != nil {
					return fail(err.Error())
				}

				p = append(p, segment{key: key})
				i = next

				continue
			}

			start := i + 1

			end := start
			for end < len(s) && s[end] >= '0' && s[end] <= '9' {
				end++
			}

			if end == start {
				return fail("index must be a non-negative number")
			}

			if end >= len(s) || s[end] != ']' {
				return fail("unterminated bracket")
			}

			n, err := strconv.Atoi(s[start:end])
			if err != nil {
				return fail("index out of range")
			}

			p = append(p, segment{index: n, isIndex: true})
			i = end + 1
		default:
			return fail("unexpected character")
		}
	}

	return p, nil
}

func parseQuotedSegment(s string, i int) (string, int, error) {
	quoted, err := strconv.QuotedPrefix(s[i+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid quoted key")
	}

	key, err := strconv.Unquote(quoted)
	if err != nil {
		return "", 0, fmt.Errorf("invalid quoted key")
	}

	if key == "" {
		return "", 0, fmt.Errorf("empty key")
	}

	end := i + 1 + len(quoted)
	if end >= len(s) || s[end] != ']' {
		return "", 0, fmt.Errorf("unterminated bracket")
	}

	return key, end + 1, nil
}

func (p woPath) String() string {
	var b strings.Builder

	for i, seg := range p {
		switch {
		case seg.isIndex:
			b.WriteString("[" + strconv.Itoa(seg.index) + "]")
		case bareKeyPattern.MatchString(seg.key):
			if i > 0 {
				b.WriteString(".")
			}

			b.WriteString(seg.key)
		default:
			b.WriteString("[" + strconv.Quote(seg.key) + "]")
		}
	}

	return b.String()
}

func lookup(doc any, p woPath) (any, bool) {
	cur := doc

	for _, seg := range p {
		if seg.isIndex {
			l, ok := cur.([]any)
			if !ok || seg.index >= len(l) {
				return nil, false
			}

			cur = l[seg.index]

			continue
		}

		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}

		v, ok := m[seg.key]
		if !ok {
			return nil, false
		}

		cur = v
	}

	if cur == nil {
		return nil, false
	}

	return cur, true
}

func deepCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			out[k] = deepCopy(x)
		}

		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = deepCopy(x)
		}

		return out
	default:
		return v
	}
}

// settable reports whether setAt can write p without leaving a partial change.
// Missing or null keys are created, so no index may follow them.
func settable(doc map[string]any, p woPath) bool {
	var cur any = doc

	for i, seg := range p {
		if seg.isIndex {
			l, ok := cur.([]any)
			if !ok || seg.index >= len(l) {
				return false
			}

			cur = l[seg.index]

			continue
		}

		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}

		next, exists := m[seg.key]
		if !exists || next == nil {
			for _, rest := range p[i+1:] {
				if rest.isIndex {
					return false
				}
			}

			return true
		}

		cur = next
	}

	return true
}

// setAt: caller passes a private copy. It is a no-op unless the whole path can be set.
func setAt(doc map[string]any, p woPath, v any) {
	if len(p) == 0 || !settable(doc, p) {
		return
	}

	var cur any = doc

	for i, seg := range p {
		last := i == len(p)-1

		if seg.isIndex {
			l, ok := cur.([]any)
			if !ok || seg.index >= len(l) {
				return
			}

			if last {
				l[seg.index] = v
				return
			}

			cur = l[seg.index]

			continue
		}

		m, ok := cur.(map[string]any)
		if !ok {
			return
		}

		if last {
			m[seg.key] = v
			return
		}

		next, exists := m[seg.key]
		if !exists || next == nil {
			next = map[string]any{}
			m[seg.key] = next
		}

		cur = next
	}
}
