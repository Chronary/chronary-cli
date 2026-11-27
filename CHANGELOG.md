# Changelog

All notable changes to `chronary-cli` will be documented in this file starting with the soft-launch release.

## 0.14.1 — unreleased

- **`auth signup --heard-about <text>` (#2651).** Optional: how you found Chronary, plain text cut to 200 characters, sent as `heard_about` on an email sign-up. The API keeps it for a new organization only and never returns it. Refused with `--anonymous`.
- **`--prevent-overlaps` on `calendars create` and `calendars update` (#595).** The calendar then refuses any event that would overlap an active event on it. `update` sends it only when the flag is given; `--prevent-overlaps=false` turns it off. `calendars get` shows `Overlaps: refused` for such a calendar, and a refused write (409) lists the conflicting event IDs.

## 0.14.0 — 2026-11-24

- **`webhooks rotate-secret` names the v2 signature headers (#2569).** Deliveries now carry `X-Signature-V2`, which also signs the delivery id, event type, test flag and replay-of id, and `X-Signature-V2-Previous` during a rotation. The help text and the "old secret keeps working" line name those headers; the current SDK verifiers check only the v2 pair. Text only; no behaviour change.
- **`-o json` / `-o yaml` print the API response as sent (#2738).** Commands decoded the body into internal structs and printed those, so fields the structs lacked were dropped (`external_sync` on `events create` / `update`, `orgId` and `deletedAt` on events, `next_cursor`, the delivery diagnostics `last_response_status` / `last_error_code` / `last_duration_ms`, newer plan limits, `payment_method` on `billing`), timestamps lost their milliseconds and `null` fields vanished. YAML printed lowercased Go field names (`orgid:`) and JSON objects such as `metadata` as lists of byte values. Both formats now carry every key and value of the REST response, with the API's key names. Output is additive except that timestamps keep the API's format.
- **`--debug` writes its HTTP traces to stderr (#2723).** They went to stdout, so `-o json --debug` was not valid JSON. The dump also redacts `setup_url` (a connection link's one-time secret).
- **Table cells are cut by display width (#2722).** Long titles, descriptions and labels were cut by bytes, so non-ASCII text was cut too early or mid-character (invalid UTF-8, sometimes a misaligned table), and short CJK titles were cut although they fit.
- **A GET refused with a `Retry-After` beyond 60 seconds is not retried (#2558).** A 429 or 503 such as an iCal sync cooldown (up to 900 s on Free) was retried twice after 5 s waits and then failed with the same error. The API's error is now shown at once. A `Retry-After` of 60 s or less is retried as before (each wait capped at 5 s). Same rule as the SDKs.
- **`--all` follows `next_cursor` (#2691).** `agents list`, `calendars list` and `events list-all` with `--all` paged by offset only, so past 10,000 rows they will fail once the API enforces its `offset` cap. They now page by the returned `next_cursor`; other lists still page by offset. `--all` also stops on an empty page instead of asking again.
- **Delete commands no longer report success when they could not ask (#2692).** With no terminal (CI, cron, containers, agent sandboxes) the confirmation prompt fails; `keys`, `agents`, `calendars`, `calendars availability-rules`, `events`, `webhooks`, `ical` and `booking-pages` `delete` used to print `Cancelled.` and exit 0 without sending the DELETE. They now exit 1 with `confirmation required ... Re-run with --yes to confirm; no request was sent`, including under `TERM=dumb` with no terminal on stdin. Answering no or Ctrl-C still prints `Cancelled.` and exits 0. Scripts should pass `--yes`.
- **`auth login` validates the key for real (#2559).** It called the unauthenticated `/health`, so a mistyped or revoked key was saved as "Authenticated." It now calls `GET /v1/usage` with the key and saves nothing when the API rejects it.
- **`auth login --profile X` no longer inherits another profile's base URL (#2527).** It used the active profile's URL, so logging in to `prod` while `dev` was active saved the prod key with the dev URL. The URL is now `--base-url` / `CHRONARY_BASE_URL`, else the default; a saved URL is never reused, so re-login repairs a profile saved with the wrong one. Validation errors name the URL used.
- **A corrupt config file is reported, never overwritten (#2562).** Commands said "no API key configured" for an unparseable `config.json`, and `auth login` then rewrote it with one profile, losing the others. Commands that need a key now name the file in an error (passing `--api-key` still works, with a warning; key-less commands such as `health` and `plans` are unaffected), and `auth login`, `switch`, `remove` and `list` leave it untouched.
- **Authentication errors keep the API's reason (#2561).** Every `authentication_error` was replaced with a re-login hint, hiding reasons such as "This key is disabled because the organization is scheduled for deletion." The reason is shown before the hint, and every API error now ends with its `request_id`.
- **`chronary usage` tells "not included" from "unlimited" (#2526).** A limit of `0` (feature not in the plan) showed as `unlimited`, and `-o json` turned the API's `null` (unlimited) into `0`. A `null` limit is now `unlimited` and stays `null` in JSON; `0` shows as `0`.
- **`chronary ical` reads the API's snake_case fields (#2521).** `list`, `get`, `create`, `update`, `pause` and `resume` showed blank agent and calendar IDs, no last-sync time or last error, and a zero created date; `-o json` emitted camelCase keys with empty values. The fields now decode, and `-o json` matches the REST response.
- **`chronary scheduling list --all` keeps `--status` and `--organizer` (#2560).** The filters were dropped on the `--all` path, so `--all --status pending` returned every proposal.

## 0.13.2 — 2026-11-22

- `chronary calendars delete` warns in its confirmation prompt that the calendar's booking pages are deleted too and their links stop working, as the API now does (#2545).
- **IDs are escaped in request paths (#2519).** An ID given as an argument or an ID flag (`--calendar`, `--agent`) now goes into the URL as one escaped segment, so a value such as `../agents/agt_1`, `whk_1#` or `evt_1?expand=all` can no longer turn `chronary events delete` or `chronary events get` into a call on another resource or add a query string. `""`, `"."` and `".."` are refused before any request is sent (`invalid ID: a path parameter must not be empty, "." or "..". No request was sent`). Real Chronary IDs are unaffected.

## 0.13.1 — 2026-11-22

- `chronary scheduling respond` prints whether the proposal auto-resolved and, when the auto-resolve failed, the API's reason with a hint to check the proposal and run `chronary scheduling resolve <id>`. The response is saved either way. `--output json` carries `auto_resolved` and `auto_resolve_error` (#2540).

## 0.13.0 — 2026-11-17

- `chronary auth signup --anonymous --agent-name <name> --tos-version <v>` signs up without an email (`POST /v1/agent/sign-up/anonymous`): a claimable org whose key works at once at Sandbox limits. It prints the key, a one-time claim URL for the person who should own the calendar, and the expiry (72 hours; unclaimed, the org and its data are deleted). `--output json` carries `claim_url` and `expires_at`. `--email` is no longer marked required, since `--anonymous` takes none; the email sign-up still refuses a missing or malformed address. The `--debug` dump redacts `claim_url` (#2335).

## 0.12.0 — 2026-11-17

- `chronary usage` shows a Calendar Connections row (live Google / Microsoft connections vs. the plan cap); `--output json` carries `calendar_connections` (#2397).
- `chronary audit-log --help` lists Developer's 30-day retention next to Free (3 days) and Pro (90 days) (#2397).

## 0.11.0 — 2026-11-16

- `events create` and `events update` take `--timezone`: a recurring series repeats on local wall time in an IANA zone (e.g. `America/Los_Angeles`), across DST changes. On update, `--timezone ""` returns the series to UTC. Send it with `--recurrence-rule` to move an existing series in one call (#2276).
- An `events update` that moves a recurring series (changes `--start`, `--recurrence-rule` or `--timezone`) keeps each cancelled occurrence on its local date. Any that no longer fit are listed after the update with the command to re-cancel them; `--output json` carries them as `droppedExdates` (#2277).
- `events get` shows the recurrence rule and time zone when set; `--output json` for events carries `timezone` and, on expanded instances, `startLocal`, `utcOffset` and `dstAdjusted`.

## 0.10.2 — 2026-11-14

- An empty `--all` list (`agents`, `calendars`, `events list` and `list-all`, `booking-pages`, `ical`, `scheduling`, `webhooks`) prints `"data": []` and the page size as `limit`, as every other list response does. It printed `"data": null` and `"limit": 0` (#2095).

## 0.10.1 — 2026-11-13

- `webhooks list --all` works again: it asked for 200 webhooks a page and the API allows at most 100. Every `--all` page size is now pinned to its endpoint's limit by a test (#1992).
- `agents create`, `calendars create`, `booking-pages create`, `events create` and `webhooks create` accept the body from `@file` or `@-` without the body flags. Without `@file` they still say which flags are missing. `events create` still needs `--calendar`. A missing or invalid `@file` now fails with the read or parse error (#1993).
- `--reminders=` (events create and update) and `--default-reminders=` (calendars create and update) send an empty list to set no reminders, as the help text said; they failed with `strconv.Atoi` before. A value that is not a whole number fails with a clear error (#1994).

## 0.10.0 — 2026-11-12

- A 429 `quota_exceeded` (a monthly plan limit) is no longer retried: the command fails at once with the quota error instead of waiting and asking again. A rate-limit 429 still retries (#1936).
- `billing` shows a scheduled plan switch (`Switching to`) and plan error hints name the plan the upgrade link leads to (`upgrade_plan`). Replay and secret rotation help text says Developer or Pro. `--output json` carries `annual_price` and the `pending_plan*` fields (#1911).

## 0.9.1 — 2026-11-09

- API keys shown by `auth status`, `auth list` (all output formats) and `--debug` now use one masked form: the key prefix and its last four characters, for example `chr_sk_...a1b2`.
- `auth signup` prints the new key once; the next-step hint no longer repeats it.
- Build with Go 1.26.10 and update `golang.org/x/text` and `golang.org/x/sys`, picking up the Go standard-library security fixes (#1642).
- `auth waitlist` prints only the API's message: the API now answers every email with the same body and no entry details. Adds `--intent` (#1655).
- `auth signup` recognises the paused-sign-ups response (`waitlisted: true`) and prints its message (#1655).

## 0.9.0 — 2026-10-19

- Add `chronary billing` (plan and billing status), `chronary billing request-upgrade [--interval] [--reason] [--wait]` and `chronary billing upgrade-request get|cancel <id>` (#1129).
- Plan and quota errors now print `Upgrade: <url>` and suggest `chronary billing request-upgrade`. Quota copy now says quotas reset on the UTC calendar month.
- Add the `org.plan_changed` webhook event type: fires once when the account's plan changes, so agents don't have to poll billing (#1130).
- Add the `calendar_connection.reauthorization_required` webhook event type: fires once when a connected human calendar stops accepting access, so agents learn their availability now fails closed (#1196).
- Add `chronary connection-links list [--status] [--calendar] [--limit]` (#1196).
- `chronary webhooks deliveries` gains `--event-type`, `--created-after`, `--created-before`, `--delivery-id`, `--test` and `--cursor`, prints the next-page cursor, and counts `dropped` deliveries (a Free-plan delivery the monthly limit stopped) (#1200).
- Add `chronary webhooks delivery <webhook-id> <delivery-id>`: one delivery and every attempt (status, duration, error) (#1200).
- Add `chronary webhooks test <webhook-id> --event <type>`: sends a signed sample event once (quota-exempt, `X-Chronary-Test: 1`) (#1200).
- Add `chronary events list-all [--calendar ...] [--agent] [--start-after] [--start-before] [--status] [--source] [--search] [--expand]`: events across all of the organization's calendars. Add `chronary calendars rotate-ical-token <id>` to regenerate a calendar's subscribe URL, and `chronary calendars list --search` (#1202).
- Add `chronary booking-pages bookings <id> [--limit] [--cursor]` (the bookings made through a page, with invitee details), `chronary booking-pages rotate-slug <id>` (replace a page's public link) and `chronary booking-pages list --calendar` (#1202).
- Add `chronary webhooks replay <webhook-id> <delivery-id>` (Pro): sends a failed or dropped delivery again as a new delivery that counts toward the monthly quota. `chronary webhooks delivery` shows `Replay of` for a replay (#588).
- Add `chronary webhooks rotate-secret <webhook-id> [--expire-old-in now|1h|24h|7d]` (Pro): prints the new signing secret once; the old one keeps signing `X-Signature-Previous` until it expires (default 24h). `webhooks get` output gains `previousSecretExpiresAt` (#594).

## 0.8.1 — 2026-07-14

- Include the `booking_pages` quota counter in `usage` JSON/table output (it was dropped). No competitor references in help text.

## 0.8.0 — 2026-07-14

- Add `booking-pages` commands (create/list/get/update/delete) for agent-created public scheduling links (#1036).

## 0.7.0 — 2026-07-12

- Add `--allow-stale` to availability commands and surface completeness/source-health fields in structured output.
- Retain connection-link create/get/cancel commands for human-approved Google/Microsoft setup.

## 0.6.1 — 2026-07-12

- Add `connection-link` commands so agents can create, inspect, and cancel human calendar setup requests.

## 0.1.3 — 2026-05-18

- Add `CONTRIBUTING.md` to the public mirror documenting that this repo is generated from a private monorepo; PRs are welcome as proof-of-concept but can't be merged directly. No behavioral change.
