# Changelog

All notable changes to `chronary-cli` will be documented in this file starting with the soft-launch release.

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
