package metrics

import "testing"

func TestStatusClass(t *testing.T) {
	cases := map[int]string{200: "2xx", 204: "2xx", 301: "3xx", 404: "4xx", 500: "5xx"}
	for code, want := range cases {
		if got := statusClass(code); got != want {
			t.Errorf("statusClass(%d) = %q, want %q", code, got, want)
		}
	}
}
