package fbproto

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessageCatalogComplete(t *testing.T) {
	const directory = "../_tools/msgs/include/firebird/impl/"
	helper, err := os.ReadFile(directory + "msg_helper.h")
	require.NoError(t, err)
	facilityDeclaration := regexp.MustCompile(`(?m)^#define FB_IMPL_MSG_FACILITY_(\w+)\s+(\d+)$`)
	facilities := make(map[string]int)
	for _, match := range facilityDeclaration.FindAllStringSubmatch(string(helper), -1) {
		id, err := strconv.Atoi(match[2])
		require.NoError(t, err)
		facilities[match[1]] = id
	}
	require.NotEmpty(t, facilities)
	all, err := os.ReadFile(directory + "msg/all.h")
	require.NoError(t, err)
	includes := regexp.MustCompile(`(?m)^#include "([^"]+)"$`).FindAllStringSubmatch(string(all), -1)
	require.NotEmpty(t, includes)
	declaration := regexp.MustCompile(`^\s*FB_IMPL_MSG(_SYMBOL|_NO_SYMBOL)?\((\w+),\s*(\d+),`)
	for _, include := range includes {
		header := include[1]
		t.Run(header, func(t *testing.T) {
			file, err := os.Open(directory + "msg/" + header)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, file.Close()) })
			scanner := bufio.NewScanner(file)
			count := 0
			for scanner.Scan() {
				match := declaration.FindStringSubmatch(scanner.Text())
				if match == nil {
					continue
				}
				facility, ok := facilities[match[2]]
				require.True(t, ok, "unknown facility %s", match[2])
				number, err := strconv.Atoi(match[3])
				require.NoError(t, err)
				code := 0x14000000 | (facility << 16) | number
				message, ok := errmsgs[code]
				if !ok {
					t.Errorf("missing %s message %d (gds code %d)", match[2], number, code)
				} else {
					require.Len(t, message.sqlstate, 5, "gds code %d", code)
					if match[1] != "" && message.sqlstate != "00000" {
						t.Errorf("gds code %d has no source SQLSTATE, want 00000, got %q", code, message.sqlstate)
					}
				}
				count++
			}
			require.NoError(t, scanner.Err())
			require.Positive(t, count, "no message declarations checked")
		})
	}
}

func TestGfixStatusVectorMessages(t *testing.T) {
	tests := []struct {
		number  int
		args    []any
		message string
		warning bool
	}{
		{24, nil, "Summary of validation errors", false},
		{55, []any{3}, "\\n\tNumber of record level errors\t: 3", false},
		{56, []any{3}, "\tNumber of Blob page errors\t: 3", false},
		{57, []any{3}, "\tNumber of data page errors\t: 3", false},
		{58, []any{3}, "\tNumber of index page errors\t: 3", false},
		{59, []any{3}, "\tNumber of pointer page errors\t: 3", false},
		{60, []any{3}, "\tNumber of transaction page errors\t: 3", false},
		{61, []any{3}, "\tNumber of database page errors\t: 3", false},
		{122, []any{3}, "\tNumber of inventory page errors\t: 3", false},
		{123, []any{3}, "\tNumber of record level warnings\t: 3", true},
		{124, []any{3}, "\tNumber of blob page warnings\t: 3", true},
		{125, []any{3}, "\tNumber of data page warnings\t: 3", true},
		{126, []any{3}, "\tNumber of index page warnings\t: 3", true},
		{127, []any{3}, "\tNumber of pointer page warnings\t: 3", true},
		{128, []any{3}, "\tNumber of transaction page warnings\t: 3", true},
		{129, []any{3}, "\tNumber of database page warnings\t: 3", true},
		{130, []any{3}, "\tNumber of inventory page warnings\t: 3", true},
		{71, []any{42}, "Transaction 42 is in limbo.", false},
		{86, []any{42}, "Could not reattach to database for transaction 42.", false},
		{91, []any{42}, "Transaction 42:", false},
		{93, []any{"host"}, "    Host Site: host", false},
		{102, []any{"database.fdb"}, "        Database Path: database.fdb", false},
		{116, []any{"password.txt", 2}, "could not open password file password.txt, errno 2", false},
		{117, []any{"password.txt", 5}, "could not read password file password.txt, errno 5", false},
		{118, []any{"password.txt"}, "empty password file password.txt", false},
		{1002, nil, "number of threads required", false},
		{1004, nil, "Configuration file name required.", false},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.number), func(t *testing.T) {
			code := 0x14030000 | tt.number
			msg := readStatusVectorTestResponse(t, func(w *XDRWriter, encoding *Encoding) {
				tag := Isc_arg_gds
				if tt.warning {
					tag = Isc_arg_warning
				}
				require.NoError(t, w.WriteInt32(tag))
				require.NoError(t, w.WriteInt32(code))
				for _, arg := range tt.args {
					switch arg := arg.(type) {
					case int:
						require.NoError(t, w.WriteInt32(Isc_arg_number))
						require.NoError(t, w.WriteInt32(arg))
					case string:
						require.NoError(t, w.WriteInt32(Isc_arg_string))
						require.NoError(t, w.WriteString(arg, encoding))
					default:
						t.Fatalf("unsupported status argument %T", arg)
					}
				}
			})
			if tt.warning {
				response, ok := msg.(*OpResponse)
				require.True(t, ok, "response type: %T", msg)
				require.False(t, response.Failure)
				require.True(t, response.Warnings)
				require.Equal(t, tt.message, response.WarnMessage)
				require.Empty(t, response.Message)
				require.Equal(t, []int{code}, response.GdsCodes)
				require.Equal(t, "HY000", response.SqlState)
			} else {
				response, ok := msg.(*ErrorResponse)
				require.True(t, ok, "validation diagnostics must remain errors: %T", msg)
				require.Equal(t, tt.message, response.Message)
				require.Equal(t, []int{code}, response.GdsCodes)
				require.Equal(t, "HY000", response.SqlState)
			}
		})
	}
}

func TestGfixSummaryPreservesSQLState(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit=%t", explicit), func(t *testing.T) {
			msg := readStatusVectorTestResponse(t, func(w *XDRWriter, encoding *Encoding) {
				for _, word := range []int{Isc_arg_gds, 335740952, Isc_arg_gds, Isc_unique_key_violation} {
					require.NoError(t, w.WriteInt32(word))
				}
				if explicit {
					require.NoError(t, w.WriteInt32(Isc_arg_sql_state))
					require.NoError(t, w.WriteString("XX001", encoding))
				}
			})
			response, ok := msg.(*ErrorResponse)
			require.True(t, ok)
			want := "23000"
			if explicit {
				want = "XX001"
			}
			require.Equal(t, want, response.SqlState)
			require.Equal(t, []int{335740952, Isc_unique_key_violation}, response.GdsCodes)
			require.Contains(t, response.Message, "Summary of validation errors")
		})
	}
}

func TestCatalogStatusVectorMessages(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		arg     string
		message string
	}{
		{"gbak unnamed", 0x140c0000, "", "could not locate appropriate error message"},
		{"gstat unnamed", 0x14150005, "5.0", "gstat version 5.0"},
		{"gstat named", 0x14150020, "", "    -u      username"},
		{"fbsvcmgr unnamed", 0x14160012, "", "Information truncated"},
		{"nbackup unnamed", 0x14180017, "backup.fbk", "PROBLEM ON \"backup.fbk\"."},
	}
	for _, tt := range tests {
		for _, warning := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/warning=%t", tt.name, warning), func(t *testing.T) {
				msg := readStatusVectorTestResponse(t, func(w *XDRWriter, encoding *Encoding) {
					tag := Isc_arg_gds
					if warning {
						require.NoError(t, w.WriteInt32(Isc_arg_gds))
						require.NoError(t, w.WriteInt32(0))
						tag = Isc_arg_warning
					}
					require.NoError(t, w.WriteInt32(tag))
					require.NoError(t, w.WriteInt32(tt.code))
					if tt.arg != "" {
						require.NoError(t, w.WriteInt32(Isc_arg_string))
						require.NoError(t, w.WriteString(tt.arg, encoding))
					}
				})
				if warning {
					response, ok := msg.(*OpResponse)
					require.True(t, ok, "response type: %T", msg)
					require.False(t, response.Failure)
					require.True(t, response.Warnings)
					require.Equal(t, tt.message, response.WarnMessage)
					require.Equal(t, []int{tt.code}, response.GdsCodes)
					require.Equal(t, "HY000", response.SqlState)
				} else {
					response, ok := msg.(*ErrorResponse)
					require.True(t, ok, "response type: %T", msg)
					require.Equal(t, tt.message, response.Message)
					require.Equal(t, []int{tt.code}, response.GdsCodes)
					require.Equal(t, "HY000", response.SqlState)
				}
			})
		}
	}
}

func readStatusVectorTestResponse(t *testing.T, writeStatus func(*XDRWriter, *Encoding)) ServerMessage {
	t.Helper()
	var input bytes.Buffer
	w := NewXDRWriter(&input)
	encoding, err := NewEncoding("UTF8")
	require.NoError(t, err)
	require.NoError(t, w.WriteInt32(Op_response))
	require.NoError(t, w.WriteInt32(0))
	require.NoError(t, w.WriteInt64(0))
	require.NoError(t, w.WriteBuffer(nil))
	writeStatus(w, encoding)
	require.NoError(t, w.WriteInt32(Isc_arg_end))
	msg, err := NewClient(&input, io.Discard, encoding).ReadSlice(0, 0)
	require.NoError(t, err)
	return msg
}
