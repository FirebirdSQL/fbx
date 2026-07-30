GO ?= go
GOFMT ?= gofmt

gen:
	$(GO) run ./_tools/msgs msg fbproto/errmsgs.go
	$(GO) run ./_tools/msgs tz fbproto/builtin_tz.go
	$(GOFMT) -w fbproto/errmsgs.go fbproto/builtin_tz.go
