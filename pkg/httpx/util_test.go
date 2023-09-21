package httpx

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestValuesFromHeader(t *testing.T) {
	type args struct {
		r           *http.Request
		header      string
		valuePrefix string
		prefixLen   int
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValuesFromHeader(tt.args.r, tt.args.header, tt.args.valuePrefix, tt.args.prefixLen)
			if !tt.wantErr(t, err, fmt.Sprintf("ValuesFromHeader(%v, %v, %v, %v)", tt.args.r, tt.args.header, tt.args.valuePrefix, tt.args.prefixLen)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ValuesFromHeader(%v, %v, %v, %v)", tt.args.r, tt.args.header, tt.args.valuePrefix, tt.args.prefixLen)
		})
	}
}

func TestValuesFromCanonical(t *testing.T) {
	type args struct {
		src   string
		deli1 string
		deli2 string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "deli token",
			args: args{
				src:   "client_token=dd;access_token=a;timestamp=1414587457;nonce=d;signature=OJZA/jnroXMK/sg3VBiUCdE4angcf9p40SmSMlwyN88=",
				deli1: ";",
				deli2: "=",
			},
			want: map[string]string{
				"client_token": "dd",
				"access_token": "a",
				"timestamp":    "1414587457",
				"nonce":        "d",
				"signature":    "OJZA/jnroXMK/sg3VBiUCdE4angcf9p40SmSMlwyN88=",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValuesFromCanonical(tt.args.src, tt.args.deli1, tt.args.deli2)
			assert.Equalf(t, tt.want, got, "ValuesFromCanonical(%v, %v, %v)", tt.args.src, tt.args.deli1, tt.args.deli2)
		})
	}
}
