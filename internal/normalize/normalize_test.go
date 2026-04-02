package normalize

import (
	"testing"
)

func TestApply_deterministic(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	input := `2024-06-01 12:00:00Z INFO req id=550e8400-e29b-41d4-a716-446655440000 ip=198.51.100.10 tok=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U key=secretval snow=1234567890123456 hash=0123456789abcdef0123456789abcdef`
	a := Apply(input, cfg)
	b := Apply(input, cfg)
	if a != b {
		t.Fatalf("non-deterministic:\n%q\n%q", a, b)
	}
	c := Apply(a, cfg)
	if c != a {
		t.Fatalf("second apply changed output:\n%q\n%q", a, c)
	}
}

func TestApply_golden(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "uuid",
			in:   `trace=550e8400-e29b-41d4-a716-446655440000`,
			want: `trace=<uuid>`,
		},
		{
			name: "ipv4",
			in:   `from 198.51.100.42`,
			want: `from <ipv4>`,
		},
		{
			name: "ipv6_full",
			in:   `peer=2001:0db8:0000:0000:0000:0000:0000:0001`,
			want: `peer=<ipv6>`,
		},
		{
			name: "ipv6_compressed",
			in:   `peer=2001:db8::1`,
			want: `peer=<ipv6>`,
		},
		{
			name: "jwt",
			in:   `auth=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U`,
			want: `auth=<token>`,
		},
		{
			name: "bearer",
			in:   `call Bearer skippy123token`,
			want: `call Bearer <token>`,
		},
		{
			name: "api_key",
			in:   `x api_key=supersecret`,
			want: `x api_key=<token>`,
		},
		{
			name: "timestamp_space",
			in:   `at 2024-06-01 12:00:00 done`,
			want: `at 2024-06-01T12:00:00Z done`,
		},
		{
			name: "unix_seconds",
			in:   `when 1715000000`,
			want: `when 2024-05-06T12:53:20Z`,
		},
		{
			name: "snowflake",
			in:   `id 1234567890123456789`,
			want: `id <num_id>`,
		},
		{
			name: "hex_commit",
			in:   `rev 0123456789abcdef0123456789abcdef`,
			want: `rev <hex_id>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Apply(tc.in, cfg)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestApply_mixedPipeline(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	in := `2024-06-01T12:00:00Z WARN user=550e8400-e29b-41d4-a716-446655440000 addr=203.0.113.5 Authorization: Bearer abcdefghijklmnop`
	want := `2024-06-01T12:00:00Z WARN user=<uuid> addr=<ipv4> Authorization: Bearer <token>`
	got := Apply(in, cfg)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApply_partialConfig(t *testing.T) {
	t.Parallel()
	in := `550e8400-e29b-41d4-a716-446655440000 198.51.100.1`
	cfg := Config{MaskUUIDs: true, MaskIPv4: true}
	got := Apply(in, cfg)
	want := `<uuid> <ipv4>`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if Apply(in, Config{}) != in {
		t.Fatal("empty config should be identity")
	}
}
