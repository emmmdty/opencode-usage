package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MiniMax reports "remaining_percent"; the derived used percent must stay
// within 0..100 no matter what the API returns (negative or >100 values
// previously leaked straight into rendering).
func TestMinimaxQueryClampsPercent(t *testing.T) {
	tests := []struct {
		name         string
		remainingPct float64
		wantUsedPct  int
	}{
		{"normal", 70, 30},
		{"negative remaining percent clamps to 100", -10, 100},
		{"overfull remaining percent clamps to 0", 150, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"base_resp":{"status_code":0},"model_remains":[{"model_name":"m","remaining_percent":%v}]}`, tt.remainingPct)
			}))
			defer srv.Close()

			usage, err := minimaxQuery("test-key", srv.URL)
			if err != nil {
				t.Fatalf("minimaxQuery failed: %v", err)
			}
			if usage.Rolling.Status != "ok" {
				t.Fatalf("expected rolling window status ok, got %q", usage.Rolling.Status)
			}
			if usage.Rolling.Percent != tt.wantUsedPct {
				t.Errorf("used percent = %d, want %d", usage.Rolling.Percent, tt.wantUsedPct)
			}
		})
	}
}

// The total/remaining fallback path is clamped the same way.
func TestMinimaxQueryClampsPercentFromTotals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"base_resp":{"status_code":0},"model_remains":[{"model_name":"m","total":100,"remaining":150}]}`)
	}))
	defer srv.Close()

	usage, err := minimaxQuery("test-key", srv.URL)
	if err != nil {
		t.Fatalf("minimaxQuery failed: %v", err)
	}
	if usage.Rolling.Percent != 0 {
		t.Errorf("used percent = %d, want 0 (remaining above total must not produce a negative percent)", usage.Rolling.Percent)
	}
}

// A percentage the API returns in an unparseable form ("abc", null, ...)
// must never be rendered as a healthy 0%; unparseable windows stay unknown
// and, when nothing usable remains, the query fails loudly.
func TestZaiGLMQueryInvalidPercentageNotFakeZero(t *testing.T) {
	t.Run("single unparseable limit fails the query", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"code":200,"success":true,"data":{"limits":[{"type":"TOKENS_LIMIT","percentage":"abc"}]}}`)
		}))
		defer srv.Close()

		usage, err := zaiGLMQuery("test-key", srv.URL)
		if err == nil {
			t.Fatalf("expected an error, got usage %+v (rolling=%+v)", usage, usage.Rolling)
		}
	})

	t.Run("mixed limits keep the valid window and mark the rest unknown", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"code":200,"success":true,"data":{"limits":[{"type":"TOKENS_LIMIT","percentage":null},{"type":"CREDIT_LIMIT","percentage":42}]}}`)
		}))
		defer srv.Close()

		usage, err := zaiGLMQuery("test-key", srv.URL)
		if err != nil {
			t.Fatalf("zaiGLMQuery failed: %v", err)
		}
		if usage.Monthly.Status != "ok" || usage.Monthly.Percent != 42 {
			t.Errorf("monthly = %+v, want ok/42", usage.Monthly)
		}
		if usage.Rolling.Status == "ok" {
			t.Errorf("rolling = %+v, want unknown status (unparseable percentage must not look like 0%%)", usage.Rolling)
		}
	})
}

// Sanity: string and fractional percentages still parse ("50", 50.5).
func TestZaiGLMQueryPercentageVariants(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{`"50"`, 50},
		{`50`, 50},
		{`50.7`, 50},
	}
	for _, tt := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{"code":200,"success":true,"data":{"limits":[{"type":"TOKENS_LIMIT","percentage":%s}]}}`, tt.raw)
		}))
		usage, err := zaiGLMQuery("test-key", srv.URL)
		srv.Close()
		if err != nil {
			t.Fatalf("raw %s: zaiGLMQuery failed: %v", tt.raw, err)
		}
		if usage.Rolling.Percent != tt.want {
			t.Errorf("raw %s: percent = %d, want %d", tt.raw, usage.Rolling.Percent, tt.want)
		}
	}
}
