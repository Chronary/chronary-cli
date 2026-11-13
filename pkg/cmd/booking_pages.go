package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/output"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newBookingPagesCmd() *cobra.Command {
	bookingPagesCmd := &cobra.Command{
		Use:     "booking-pages",
		Aliases: []string{"booking-page", "booking", "bkp"},
		Short:   "Manage public booking pages (agent-created scheduling links)",
	}

	bookingPagesCmd.AddCommand(newBookingPagesListCmd())
	bookingPagesCmd.AddCommand(newBookingPagesCreateCmd())
	bookingPagesCmd.AddCommand(newBookingPagesGetCmd())
	bookingPagesCmd.AddCommand(newBookingPagesUpdateCmd())
	bookingPagesCmd.AddCommand(newBookingPagesDeleteCmd())
	bookingPagesCmd.AddCommand(newBookingPagesBookingsCmd())
	bookingPagesCmd.AddCommand(newBookingPagesRotateSlugCmd())

	return bookingPagesCmd
}

func newBookingPagesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List booking pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			path := "/v1/booking-pages"
			if v, _ := cmd.Flags().GetString("calendar"); v != "" {
				path = appendQueryParams(path, url.Values{"calendar_id": []string{v}})
			}
			all, _ := cmd.Flags().GetBool("all")

			var body []byte
			if all {
				items, total, err := fetchAllPages(c, path, pageSizeBookingPages)
				if err != nil {
					return formatError(err)
				}
				body, err = rewrapList(items, total)
				if err != nil {
					return fmt.Errorf("building response: %w", err)
				}
			} else {
				params := url.Values{}
				if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
					params.Set("limit", strconv.Itoa(v))
				}
				if v, _ := cmd.Flags().GetInt("offset"); v > 0 {
					params.Set("offset", strconv.Itoa(v))
				}
				path = appendQueryParams(path, params)
				var fetchErr error
				body, _, fetchErr = c.Get(path)
				if fetchErr != nil {
					return formatError(fetchErr)
				}
			}

			var resp client.ListResponse[client.BookingPage]
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, p := range resp.Data {
				active := "yes"
				if !p.Active {
					active = "no"
				}
				rows[i] = []string{
					p.ID,
					p.Title,
					fmt.Sprintf("%d min", p.DurationMinutes),
					active,
					strconv.Itoa(p.BookingsCount),
					p.BookingURL,
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"ID", "Title", "Duration", "Active", "Bookings", "URL"},
				Rows:    rows,
			}, nc)

			fmt.Printf("\nShowing %d of %d booking pages\n", len(resp.Data), resp.Total)
			return nil
		},
	}

	cmd.Flags().Int("limit", 50, "Max results to return")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")
	cmd.Flags().String("calendar", "", "Only pages bound to this calendar ID")

	return cmd
}

func newBookingPagesBookingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookings <id>",
		Short: "List the bookings made through a booking page (newest first)",
		Long: "Lists the events booked through the page with the invitee's name and email, newest start first. " +
			"Pass the printed next cursor with --cursor for the next page. " +
			"Bookings made before this command's API endpoint was released are not linked to their page.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			params := url.Values{}
			if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
				params.Set("limit", strconv.Itoa(v))
			}
			if v, _ := cmd.Flags().GetString("cursor"); v != "" {
				params.Set("cursor", v)
			}
			path := "/v1/booking-pages/" + args[0] + "/bookings"
			path = appendQueryParams(path, params)
			body, _, err := c.Get(path)
			if err != nil {
				return formatError(err)
			}

			var resp client.BookingList
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, resp) {
				return nil
			}

			nc := noColor(cmd)
			rows := make([][]string, len(resp.Data))
			for i, b := range resp.Data {
				rows[i] = []string{
					b.ID,
					b.StartTime.Format("2006-01-02 15:04 MST"),
					b.Status,
					derefOr(b.InviteeName, "-"),
					derefOr(b.InviteeEmail, "-"),
				}
			}

			output.RenderTable(output.TableDef{
				Headers: []string{"Event ID", "Start", "Status", "Invitee", "Email"},
				Rows:    rows,
			}, nc)

			if resp.NextCursor != nil {
				fmt.Printf("\nMore bookings: --cursor %s\n", *resp.NextCursor)
			}
			return nil
		},
	}

	cmd.Flags().Int("limit", 50, "Max results to return (1-100)")
	cmd.Flags().String("cursor", "", "The next cursor printed by the previous page")

	return cmd
}

func newBookingPagesRotateSlugCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate-slug <id>",
		Short: "Replace a booking page's public link (the old link stops working)",
		Long: "Gives the booking page a new slug and prints its new booking URL. " +
			"The old /book/<slug> link stops working at once; existing bookings are unaffected.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Post("/v1/booking-pages/"+args[0]+"/rotate-slug", nil)
			if err != nil {
				return formatError(err)
			}

			var page client.BookingPage
			if err := json.Unmarshal(body, &page); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, page) {
				return nil
			}

			fmt.Printf("Link rotated for %s\n", page.ID)
			fmt.Printf("URL: %s\n", page.BookingURL)
			return nil
		},
	}
}

func derefOr(s *string, fallback string) string {
	if s == nil || *s == "" {
		return fallback
	}
	return *s
}

func newBookingPagesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [@file]",
		Short: "Create a booking page",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload, err := checkFileArg(args, 0)
			if err != nil {
				return err
			}
			if payload == nil {
				if err := requireFlagsSet(cmd, "calendar", "title"); err != nil {
					return err
				}
				calendarID, _ := cmd.Flags().GetString("calendar")
				title, _ := cmd.Flags().GetString("title")
				p := map[string]any{
					"calendar_id": calendarID,
					"title":       title,
				}
				if cmd.Flags().Changed("description") {
					v, _ := cmd.Flags().GetString("description")
					p["description"] = v
				}
				if cmd.Flags().Changed("duration") {
					v, _ := cmd.Flags().GetInt("duration")
					p["duration_minutes"] = v
				}
				if cmd.Flags().Changed("buffer") {
					v, _ := cmd.Flags().GetInt("buffer")
					p["buffer_minutes"] = v
				}
				if cmd.Flags().Changed("window-days") {
					v, _ := cmd.Flags().GetInt("window-days")
					p["window_days"] = v
				}
				if cmd.Flags().Changed("min-notice") {
					v, _ := cmd.Flags().GetInt("min-notice")
					p["min_notice_minutes"] = v
				}
				if cmd.Flags().Changed("timezone") {
					v, _ := cmd.Flags().GetString("timezone")
					p["timezone"] = v
				}
				payload = p
			}

			body, _, err := c.Post("/v1/booking-pages", payload)
			if err != nil {
				return formatError(err)
			}

			var page client.BookingPage
			if err := json.Unmarshal(body, &page); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, page) {
				return nil
			}

			fmt.Printf("Created booking page %s (%s)\n", page.ID, page.Title)
			fmt.Printf("Share this URL: %s\n", page.BookingURL)
			return nil
		},
	}

	cmd.Flags().String("calendar", "", "Calendar ID the booking resolves to (required unless using @file)")
	cmd.Flags().String("title", "", "Booking page title (required unless using @file)")
	cmd.Flags().String("description", "", "Description shown to the booker")
	cmd.Flags().Int("duration", 30, "Slot length in minutes")
	cmd.Flags().Int("buffer", 0, "Buffer minutes before/after existing events")
	cmd.Flags().Int("window-days", 14, "How far ahead bookings are allowed")
	cmd.Flags().Int("min-notice", 0, "Minimum lead time (minutes) before a bookable slot")
	cmd.Flags().String("timezone", "UTC", "IANA timezone for display + working hours")

	return cmd
}

func newBookingPagesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a booking page by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			body, _, err := c.Get("/v1/booking-pages/" + args[0])
			if err != nil {
				return formatError(err)
			}

			var page client.BookingPage
			if err := json.Unmarshal(body, &page); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, page) {
				return nil
			}

			fmt.Printf("ID:        %s\n", page.ID)
			fmt.Printf("Title:     %s\n", page.Title)
			fmt.Printf("Calendar:  %s\n", page.CalendarID)
			fmt.Printf("Duration:  %d min\n", page.DurationMinutes)
			fmt.Printf("Window:    %d days\n", page.WindowDays)
			fmt.Printf("Timezone:  %s\n", page.Timezone)
			fmt.Printf("Active:    %t\n", page.Active)
			fmt.Printf("Bookings:  %d\n", page.BookingsCount)
			fmt.Printf("URL:       %s\n", page.BookingURL)
			return nil
		},
	}
}

func newBookingPagesUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a booking page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			payload := map[string]any{}
			if cmd.Flags().Changed("title") {
				v, _ := cmd.Flags().GetString("title")
				payload["title"] = v
			}
			if cmd.Flags().Changed("description") {
				v, _ := cmd.Flags().GetString("description")
				payload["description"] = v
			}
			if cmd.Flags().Changed("duration") {
				v, _ := cmd.Flags().GetInt("duration")
				payload["duration_minutes"] = v
			}
			if cmd.Flags().Changed("buffer") {
				v, _ := cmd.Flags().GetInt("buffer")
				payload["buffer_minutes"] = v
			}
			if cmd.Flags().Changed("window-days") {
				v, _ := cmd.Flags().GetInt("window-days")
				payload["window_days"] = v
			}
			if cmd.Flags().Changed("min-notice") {
				v, _ := cmd.Flags().GetInt("min-notice")
				payload["min_notice_minutes"] = v
			}
			if cmd.Flags().Changed("timezone") {
				v, _ := cmd.Flags().GetString("timezone")
				payload["timezone"] = v
			}
			if cmd.Flags().Changed("active") {
				v, _ := cmd.Flags().GetBool("active")
				payload["active"] = v
			}

			if len(payload) == 0 {
				return fmt.Errorf("at least one flag required: --title, --description, --duration, --buffer, --window-days, --min-notice, --timezone, --active")
			}

			body, _, err := c.Patch("/v1/booking-pages/"+args[0], payload)
			if err != nil {
				return formatError(err)
			}

			var page client.BookingPage
			if err := json.Unmarshal(body, &page); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if printStructured(cmd, page) {
				return nil
			}

			fmt.Printf("Updated booking page %s (%s)\n", page.ID, page.Title)
			return nil
		},
	}

	cmd.Flags().String("title", "", "New title")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().Int("duration", 0, "New slot length in minutes")
	cmd.Flags().Int("buffer", 0, "New buffer minutes")
	cmd.Flags().Int("window-days", 0, "New window in days")
	cmd.Flags().Int("min-notice", 0, "New minimum notice in minutes")
	cmd.Flags().String("timezone", "", "New timezone")
	cmd.Flags().Bool("active", true, "Whether the page accepts bookings")

	return cmd
}

func newBookingPagesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete (deactivate) a booking page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if err := requireAPIKey(c); err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			yes, _ := cmd.Flags().GetBool("yes")
			if !force && !yes {
				var confirm bool
				err := huh.NewConfirm().
					Title(fmt.Sprintf("Delete booking page %s?", args[0])).
					Description("Its hosted URL will stop resolving. Already-booked events are unaffected.").
					Value(&confirm).
					Run()
				if err != nil || !confirm {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			_, _, err = c.Delete("/v1/booking-pages/" + args[0])
			if err != nil {
				return formatError(err)
			}

			fmt.Printf("Booking page %s deleted.\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	return cmd
}
