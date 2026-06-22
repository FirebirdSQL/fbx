package fbconn_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"rdb.red-soft.ru/fbx/fbconn"
)

func TestConfigError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "url with password",
			err:         fbconn.NewParseConfigError("firebirdsql://foo:password@host", "msg", nil),
			expectedMsg: "cannot parse `firebirdsql://foo:xxxxx@host`: msg",
		},
		{
			name:        "keyword/value with password unquoted",
			err:         fbconn.NewParseConfigError("host=host password=password user=user", "msg", nil),
			expectedMsg: "cannot parse `host=host password=xxxxx user=user`: msg",
		},
		{
			name:        "keyword/value with password quoted",
			err:         fbconn.NewParseConfigError("host=host password='pass word' user=user", "msg", nil),
			expectedMsg: "cannot parse `host=host password=xxxxx user=user`: msg",
		},
		{
			name:        "weird url",
			err:         fbconn.NewParseConfigError("firebirdsql://foo::password@host:1:", "msg", nil),
			expectedMsg: "cannot parse `firebirdsql://foo:xxxxx@host:1:`: msg",
		},
		{
			name:        "weird url with slash in password",
			err:         fbconn.NewParseConfigError("firebirdsql://user:pass/word@host:3050/db_name", "msg", nil),
			expectedMsg: "cannot parse `firebirdsql://user:xxxxx@host:3050/db_name`: msg",
		},
		{
			name:        "url without password",
			err:         fbconn.NewParseConfigError("firebirdsql://other@host/db", "msg", nil),
			expectedMsg: "cannot parse `firebirdsql://other@host/db`: msg",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.EqualError(t, tt.err, tt.expectedMsg)
		})
	}
}
