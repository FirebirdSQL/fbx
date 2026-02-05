package fbx

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// NamedArgs can be used as the first argument to a query method. It will replace every '@' named placeholder with a '?'
// ordinal placeholder and construct the appropriate arguments.
//
// For example, the following two queries are equivalent:
//
//	conn.Query(ctx, "select * from widgets where foo = @foo and bar = @bar", fbx.NamedArgs{"foo": 1, "bar": 2})
//	conn.Query(ctx, "select * from widgets where foo = ? and bar = ?", 1, 2)
//
// Named placeholders are case sensitive and must start with a letter or underscore. Subsequent characters can be
// letters, numbers, or underscores.
//
// Note that Firebird arguments not ordinal, but positional, final arguments count can increase after query rewriting if
// named argument used in query multiple times
type NamedArgs map[string]any

// RewriteQuery implements the QueryRewriter interface.
func (na NamedArgs) RewriteQuery(ctx context.Context, conn *Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
	return rewriteQuery(na, sql, false)
}

// StrictNamedArgs can be used in the same way as NamedArgs, but provided arguments are also checked to include all
// named arguments that the sql query uses, and no extra arguments.
type StrictNamedArgs map[string]any

// RewriteQuery implements the QueryRewriter interface.
func (sna StrictNamedArgs) RewriteQuery(ctx context.Context, conn *Conn, sql string, args []any) (newSQL string, newArgs []any, err error) {
	return rewriteQuery(sna, sql, true)
}

type namedArg string

type sqlLexer struct {
	src     string
	start   int
	pos     int
	nested  int // multiline comment nesting level.
	stateFn stateFn
	parts   []any

	namedArgs        map[namedArg]struct{}
	positionalToName []namedArg
}

type stateFn func(*sqlLexer) stateFn

func rewriteQuery(na map[string]any, sql string, isStrict bool) (newSQL string, newArgs []any, err error) {
	l := &sqlLexer{
		src:              sql,
		stateFn:          rawState,
		namedArgs:        make(map[namedArg]struct{}, len(na)),
		positionalToName: make([]namedArg, 0, len(na)),
	}

	for l.stateFn != nil {
		l.stateFn = l.stateFn(l)
	}

	sb := strings.Builder{}
	for _, p := range l.parts {
		switch p := p.(type) {
		case string:
			sb.WriteString(p)
		case namedArg:
			sb.WriteRune('?')
		}
	}

	newArgs = make([]any, len(l.positionalToName))
	for pos, name := range l.positionalToName {
		var found bool
		newArgs[pos], found = na[string(name)]
		if isStrict && !found {
			return "", nil, fmt.Errorf("argument %s found in sql query but not present in StrictNamedArgs", name)
		}
	}

	if isStrict {
		for name := range na {
			if _, found := l.namedArgs[namedArg(name)]; !found {
				return "", nil, fmt.Errorf("argument %s of StrictNamedArgs not found in sql query", name)
			}
		}
	}

	return sb.String(), newArgs, nil
}

func rawState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case 'e', 'E':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune == '\'' {
				l.pos += width
				return escapeStringState
			}
		case '\'':
			return singleQuoteState
		case '"':
			return doubleQuoteState
		case '@':
			nextRune, _ := utf8.DecodeRuneInString(l.src[l.pos:])
			if isLetter(nextRune) || nextRune == '_' {
				if l.pos-l.start > 0 {
					l.parts = append(l.parts, l.src[l.start:l.pos-width])
				}
				l.start = l.pos
				return namedArgState
			}
		case '-':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune == '-' {
				l.pos += width
				return oneLineCommentState
			}
		case '/':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune == '*' {
				l.pos += width
				return multilineCommentState
			}
		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func namedArgState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		if r == utf8.RuneError {
			if l.pos-l.start > 0 {
				na := namedArg(l.src[l.start:l.pos])
				l.positionalToName = append(l.positionalToName, na)
				if _, found := l.namedArgs[na]; !found {
					l.namedArgs[na] = struct{}{}
				}
				l.parts = append(l.parts, na)
				l.start = l.pos
			}
			return nil
		} else if !(isLetter(r) || (r >= '0' && r <= '9') || r == '_') {
			l.pos -= width
			na := namedArg(l.src[l.start:l.pos])
			l.positionalToName = append(l.positionalToName, na)
			if _, found := l.namedArgs[na]; !found {
				l.namedArgs[na] = struct{}{}
			}
			l.parts = append(l.parts, namedArg(na))
			l.start = l.pos
			return rawState
		}
	}
}

func singleQuoteState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case '\'':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune != '\'' {
				return rawState
			}
			l.pos += width
		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}

func doubleQuoteState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case '"':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune != '"' {
				return rawState
			}
			l.pos += width
		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}

func escapeStringState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case '\\':
			_, width = utf8.DecodeRuneInString(l.src[l.pos:])
			l.pos += width
		case '\'':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune != '\'' {
				return rawState
			}
			l.pos += width
		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}

func oneLineCommentState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case '\\':
			_, width = utf8.DecodeRuneInString(l.src[l.pos:])
			l.pos += width
		case '\n', '\r':
			return rawState
		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}

func multilineCommentState(l *sqlLexer) stateFn {
	for {
		r, width := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += width

		switch r {
		case '/':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune == '*' {
				l.pos += width
				l.nested++
			}
		case '*':
			nextRune, width := utf8.DecodeRuneInString(l.src[l.pos:])
			if nextRune != '/' {
				continue
			}

			l.pos += width
			if l.nested == 0 {
				return rawState
			}
			l.nested--

		case utf8.RuneError:
			if l.pos-l.start > 0 {
				l.parts = append(l.parts, l.src[l.start:l.pos])
				l.start = l.pos
			}
			return nil
		}
	}
}
