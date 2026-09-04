package pagination

import (
	"errors"
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		query, err := Parse(url.Values{})
		if err != nil {
			t.Fatal(err)
		}
		if query.Page != DefaultPage || query.Size != DefaultSize {
			t.Fatalf("unexpected defaults: %+v", query)
		}
	})

	t.Run("accepts valid values", func(t *testing.T) {
		query, err := Parse(url.Values{"page": {"3"}, "size": {"25"}})
		if err != nil {
			t.Fatal(err)
		}
		if query.Page != 3 || query.Size != 25 {
			t.Fatalf("unexpected query: %+v", query)
		}
	})

	for _, values := range []url.Values{
		{"page": {"0"}},
		{"page": {"invalid"}},
		{"size": {"0"}},
		{"size": {"101"}},
	} {
		if _, err := Parse(values); !errors.Is(err, ErrQuery) {
			t.Fatalf("expected ErrQuery for %v, got %v", values, err)
		}
	}
}

func TestPageMath(t *testing.T) {
	query := Clamp(Query{Page: 9, Size: 10}, 21)
	if query.Page != 3 || Offset(query) != 20 || TotalPages(21, 10) != 3 {
		t.Fatalf("unexpected page result: %+v", query)
	}

	empty := Clamp(Query{Page: 9, Size: 10}, 0)
	if empty.Page != 1 || Offset(empty) != 0 || TotalPages(0, 10) != 0 {
		t.Fatalf("unexpected empty result: %+v", empty)
	}
}
