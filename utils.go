package fbx

import "strings"

func IsSqlSearchPathChanged(sql string) bool {
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
			reset_symbol()
		case '/':
			if !commentTracker.lineOpend {
				if !commentTracker.blockStart && !commentTracker.blockEnd {
					commentTracker.blockStart = true
				} else if commentTracker.blockEnd {
					commentTracker.blockEnd = false
					commentTracker.blockOpend = false
				}
			}
		case '*':
			if !commentTracker.lineOpend {
				if !commentTracker.blockEnd && !commentTracker.blockStart {
					commentTracker.blockEnd = true
				} else if commentTracker.blockStart {
					commentTracker.blockStart = false
					commentTracker.blockOpend = true
					reset_symbol()
				}
			}
		case '"':
			if !commentTracker.lineOpend && !commentTracker.blockOpend {
				isDoubleOpend = !isDoubleOpend
			}
		case '\'':
			if !commentTracker.lineOpend && !commentTracker.blockOpend {
				isSingleOpend = !isSingleOpend
			}
		case '-':
			if !commentTracker.blockOpend {
				if !commentTracker.lineStart {
					commentTracker.lineStart = true
				} else {
					commentTracker.lineStart = false
					commentTracker.lineOpend = true
					reset_symbol()
				}
			}
		case '\n', '\r':
			reset_symbol()
			commentTracker.lineOpend = false
		default:
			if !isSingleOpend && !isDoubleOpend && !commentTracker.blockOpend && !commentTracker.lineOpend {
				currentSymbol.WriteRune(c)
			}
			commentTracker.blockEnd = false
		}
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
