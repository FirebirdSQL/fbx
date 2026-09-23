package fbx

import "strings"

func containsSearchPathChangingInSql(sql string) bool {
	currentSymbol := strings.Builder{}
	currentSymbol.Grow(15)

	symbols := make([]string, 0, 3)
	commentTracker := struct {
		blockStart bool
		blockEnd   bool
		blockOpend bool
		lineStart  bool
		lineOpend  bool
	}{}
	isSingleOpend := false
	isDoubleOpend := false

	reset_symbol := func() {
		commentTracker.blockStart = false
		commentTracker.blockEnd = false
		isSingleOpend = false
		isDoubleOpend = false
		if currentSymbol.Len() > 0 {
			symbols = append(symbols, strings.ToLower(currentSymbol.String()))
		}
		currentSymbol.Reset()
	}
	for _, c := range sql {
		switch c {
		case ' ', '\t', '\f', '\v', ';':
		case '/':
			if !commentTracker.lineOpend {
				if commentTracker.blockEnd {
					commentTracker.blockEnd = false
					commentTracker.blockOpend = false
				} else if !commentTracker.blockOpend {
					commentTracker.blockStart = true
				}
			}
			continue
		case '*':
			if !commentTracker.lineOpend {
				if commentTracker.blockOpend {
					commentTracker.blockEnd = true
					continue
				} else if commentTracker.blockStart {
					commentTracker.blockStart = false
					commentTracker.blockOpend = true
				}
			}
		case '"':
			if !commentTracker.lineOpend && !commentTracker.blockOpend {
				isDoubleOpend = !isDoubleOpend
				continue
			}
		case '\'':
			if !commentTracker.lineOpend && !commentTracker.blockOpend {
				isSingleOpend = !isSingleOpend
				continue
			}
		case '-':
			if !commentTracker.lineOpend && !commentTracker.blockOpend {
				if !commentTracker.lineStart {
					commentTracker.lineStart = true

					continue
				} else {
					commentTracker.lineStart = false
					commentTracker.lineOpend = true
				}
			}
		case '\n', '\r':
			commentTracker.lineOpend = false
		default:
			if !isSingleOpend && !isDoubleOpend && !commentTracker.blockOpend && !commentTracker.lineOpend {
				currentSymbol.WriteRune(c)
			}
			commentTracker.blockEnd = false
			continue
		}
		reset_symbol()
		if len(symbols) == 1 && symbols[0] != "set" && symbols[0] != "alter" {
			break
		}
		if len(symbols) == 3 {
			break
		}

	}
	if currentSymbol.Len() > 0 {
		symbols = append(symbols, strings.ToLower(currentSymbol.String()))
	}

	if len(symbols) >= 3 && (symbols[0] == "set" && symbols[1] == "search_path" && symbols[2] == "to" ||
		symbols[0] == "alter" && symbols[1] == "session" && symbols[2] == "reset") {
		return true
	}

	return false
}
