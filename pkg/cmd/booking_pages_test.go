package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #1202: booking page calendar filter, bookings list and link rotation.

func testBookingPage(slug string) map[string]any {
	return map[string]any{
		"id": "bkp_1", "org_id": "org_1", "calendar_id": "cal_1", "agent_id": nil,
		"slug": slug, "title": "Intro", "description": nil,
		"duration_minutes": 30, "buffer_minutes": 0, "window_days": 14, "min_notice_minutes": 0,
		"timezone": "UTC", "availability_constraints": nil, "active": true, "bookings_count": 1,
		"booking_url": "https://api.chronary.ai/book/" + slug,
		"created_at":  "2026-07-14T10:00:00Z", "updated_at": "2026-07-14T10:00:00Z",
	}
}

func TestBookingPagesListCalendarFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/booking-pages", r.URL.Path)
		assert.Equal(t, "cal_1", r.URL.Query().Get("calendar_id"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		json.NewEncoder(w).Encode(map[string]any{"data": []any{testBookingPage("s1")}, "total": 1, "limit": 10, "offset": 0})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"booking-pages", "list", "--calendar", "cal_1", "--limit", "10", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestBookingPagesBookings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/booking-pages/bkp_1/bookings", r.URL.Path)
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "abc", r.URL.Query().Get("cursor"))
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{
				"id": "evt_1", "booking_page_id": "bkp_1", "calendar_id": "cal_1",
				"start_time": "2026-07-15T16:00:00Z", "end_time": "2026-07-15T16:30:00Z",
				"status": "confirmed", "invitee_name": "Ada", "invitee_email": "ada@example.com",
				"invitee_notes": nil, "created_at": "2026-07-14T10:00:00Z",
			}},
			"next_cursor": "next",
		})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"booking-pages", "bookings", "bkp_1", "--limit", "5", "--cursor", "abc", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestBookingPagesBookingsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("cursor"))
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "next_cursor": nil})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"booking-pages", "bookings", "bkp_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL, "--output", "json"})
	require.NoError(t, rootCmd.Execute())
}

func TestBookingPagesRotateSlug(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/booking-pages/bkp_1/rotate-slug", r.URL.Path)
		json.NewEncoder(w).Encode(testBookingPage("newslug"))
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"booking-pages", "rotate-slug", "bkp_1", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	require.NoError(t, rootCmd.Execute())
}

func TestBookingPagesRotateSlugNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"type": "not_found", "message": "Booking page not found"}})
	}))
	defer srv.Close()

	rootCmd := NewRootCmd("test")
	rootCmd.SetArgs([]string{"booking-pages", "rotate-slug", "bkp_x", "--api-key", "chr_sk_xxx", "--base-url", srv.URL})
	assert.Error(t, rootCmd.Execute())
}

func TestDerefOr(t *testing.T) {
	empty := ""
	name := "Ada"
	assert.Equal(t, "-", derefOr(nil, "-"))
	assert.Equal(t, "-", derefOr(&empty, "-"))
	assert.Equal(t, "Ada", derefOr(&name, "-"))
}
