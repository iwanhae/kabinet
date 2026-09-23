package schema

import (
	"testing"
	"time"
)

func TestResolveTimestampPrecedence(t *testing.T) {
	creation := time.Unix(100, 0)
	first := creation.Add(time.Second)
	last := first.Add(time.Second)
	series := last.Add(time.Second)

	for _, tc := range []struct {
		name                          string
		series, last, first, creation *time.Time
		want                          time.Time
		ok                            bool
	}{
		{name: "series", series: &series, last: &last, first: &first, creation: &creation, want: series, ok: true},
		{name: "last", last: &last, first: &first, creation: &creation, want: last, ok: true},
		{name: "first", first: &first, creation: &creation, want: first, ok: true},
		{name: "creation", creation: &creation, want: creation, ok: true},
		{name: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveTimestamp(tc.series, tc.last, tc.first, tc.creation)
			if ok != tc.ok || !got.Equal(tc.want) {
				t.Fatalf("ResolveTimestamp()=(%s,%v), want (%s,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestTimestampProjectionUsesSingleExpression(t *testing.T) {
	if Columns[7].Name != "timestamp" || Columns[7].rawExpr != TimestampSQLExpr {
		t.Fatalf("timestamp column does not use canonical expression: %+v", Columns[7])
	}
	for _, column := range Columns {
		if (column.Name == "firstTimestamp" || column.Name == "lastTimestamp") && column.rawExpr != "" {
			t.Fatalf("source timestamp field %s must remain unmodified: %s", column.Name, column.rawExpr)
		}
	}
}
