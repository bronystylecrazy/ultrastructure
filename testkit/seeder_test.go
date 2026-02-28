package testkit

import (
	"reflect"
	"testing"
)

func TestInsertTablesFromSQL(t *testing.T) {
	sqlText := `
		INSERT INTO customers (id, name) VALUES ('cust-1', 'Alice');
		insert into products (id, sku) values ('prod-1', 'SKU-1');
		INSERT INTO public.audit_logs (id) VALUES ('a1');
		INSERT INTO "weird_table" (id) VALUES ('x');
		INSERT INTO public.audit_logs (id) VALUES ('a2');
	`

	got := insertTablesFromSQL(sqlText)
	want := []string{"customers", "products", "public.audit_logs", `"weird_table"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tables: got=%v want=%v", got, want)
	}
}

func TestInsertTablesFromSQLEmpty(t *testing.T) {
	got := insertTablesFromSQL(`SELECT 1;`)
	if got != nil {
		t.Fatalf("expected nil tables, got=%v", got)
	}
}

func TestUniqueStrings(t *testing.T) {
	got := uniqueStrings([]string{"a", "b", "a", "c", "b"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected unique values: got=%v want=%v", got, want)
	}
}
