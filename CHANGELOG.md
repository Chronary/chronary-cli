# Changelog

All notable changes to `chronary-cli` will be documented in this file starting with the soft-launch release.

## 0.10.2 — Unreleased

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
