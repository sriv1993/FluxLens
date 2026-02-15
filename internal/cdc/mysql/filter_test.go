package mysql

import "testing"

func TestTableFilter(t *testing.T) {
	f := newTableFilter([]string{"shop.orders", "Shop.ITEMS"})
	if !f.allow("shop", "orders") {
		t.Fatal("expected shop.orders")
	}
	if !f.allow("Shop", "ITEMS") {
		t.Fatal("expected shop.items case insensitive")
	}
	if f.allow("shop", "users") {
		t.Fatal("expected shop.users denied")
	}
	if f.allow("other", "orders") {
		t.Fatal("expected other.orders denied")
	}

	allowAll := newTableFilter(nil)
	if !allowAll.allow("any", "table") {
		t.Fatal("empty filter should allow all")
	}
}
