package mysql

import "testing"

func TestParseDSN(t *testing.T) {
	info, err := parseDSN("user:pass@tcp(localhost:3307)/appdb?parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	if info.Host != "localhost" || info.Port != 3307 {
		t.Fatalf("addr %+v", info)
	}
	if info.User != "user" || info.Password != "pass" || info.Database != "appdb" {
		t.Fatalf("creds/db %+v", info)
	}
}
