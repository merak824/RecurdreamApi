package admin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSnapshotGranularity(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "hour passthrough", input: "hour", want: "hour"},
		{name: "day passthrough", input: "day", want: "day"},
		{name: "week passthrough", input: "week", want: "week"},
		{name: "month passthrough", input: "month", want: "month"},
		{name: "mixed case", input: "Week", want: "week"},
		{name: "padded", input: "  MONTH  ", want: "month"},
		{name: "empty falls back to day", input: "", want: "day"},
		{name: "unknown falls back to day", input: "quarter", want: "day"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeSnapshotGranularity(tc.input))
		})
	}
}
