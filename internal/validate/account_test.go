package validate

import "testing"

func TestSystemUser(t *testing.T) {
	for _, value := range []string{"root", "wcp_123", "wcp_0123456789ab-extra", "WCP_0123456789AB"} {
		if err := SystemUser(value); err == nil {
			t.Fatalf("SystemUser(%q) should fail", value)
		}
	}
	if err := SystemUser("wcp_0123456789ab"); err != nil {
		t.Fatal(err)
	}
}

func TestID(t *testing.T) {
	if err := ID("accountId", "0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := ID("accountId", "account-1"); err == nil {
		t.Fatal("expected invalid ID error")
	}
}
