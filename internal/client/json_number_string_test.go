package client

import (
	"encoding/json"
	"testing"
)

func TestJSONNumberString_Unmarshal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "integer", in: `4`, want: "4"},
		{name: "string", in: `"10"`, want: "10"},
		{name: "null", in: `null`, want: ""},
		{name: "float", in: `4.5`, wantErr: true},
		{name: "object", in: `{}`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got JSONNumberString
			err := json.Unmarshal([]byte(tc.in), &got)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != tc.want {
				t.Fatalf("got %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestApplication_BasicAuthCostsAcceptIntegerJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"uuid":"app-1","http_basic_auth_argon2id_time_cost":4,"http_basic_auth_bcrypt_cost":10}`)
	var app Application
	if err := json.Unmarshal(raw, &app); err != nil {
		t.Fatal(err)
	}
	if app.HTTPBasicAuthArgon2idTimeCost.String() != "4" {
		t.Fatalf("time cost %q", app.HTTPBasicAuthArgon2idTimeCost)
	}
	if app.HTTPBasicAuthBcryptCost.String() != "10" {
		t.Fatalf("bcrypt cost %q", app.HTTPBasicAuthBcryptCost)
	}

	raw = []byte(`{"http_basic_auth_argon2id_time_cost":"2","http_basic_auth_bcrypt_cost":"12"}`)
	app = Application{}
	if err := json.Unmarshal(raw, &app); err != nil {
		t.Fatal(err)
	}
	if app.HTTPBasicAuthArgon2idTimeCost.String() != "2" || app.HTTPBasicAuthBcryptCost.String() != "12" {
		t.Fatalf("string form time=%q bcrypt=%q", app.HTTPBasicAuthArgon2idTimeCost, app.HTTPBasicAuthBcryptCost)
	}
}
