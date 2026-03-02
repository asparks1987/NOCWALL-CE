# CE Feature Pack Test Matrix

Validation reference for CE adoption features `CEF01` through `CEF25`.
Use with `scripts/ce-feature-pack-smoke.sh` for API smoke coverage and manual UI checks for wallboard flows.

## Automated Smoke Coverage

Run:

```bash
BASE_URL=http://localhost DRY_RUN=0 ./scripts/ce-feature-pack-smoke.sh
```

Covered endpoint flows:
- account register/login
- demo mode toggle (`demo_mode_get`, `demo_mode_set`)
- preferences read/write (`prefs_get`, `prefs_save`)
- source diagnostics (`sources_diagnostics`)
- in-app changelog payload (`whats_new`)

## Manual CE UI Matrix

1. `CEF01` First-run setup wizard: verify wizard appears when account has no sources and can save/test source.
2. `CEF02` Demo data toggle: enable/disable in dashboard/settings and verify device payload switches.
3. `CEF03` Source diagnostics panel: run diagnostics and verify DNS/TLS/API fields update.
4. `CEF04` Poll Now per source: trigger per-source poll and confirm refreshed status.
5. `CEF05` Source status strip: verify poll health, latency, last poll timestamps.
6. `CEF06` Global search: verify name/host/mac/site queries filter cards.
7. `CEF07` Quick filters: verify all/online/offline tabs filter device list.
8. `CEF08` Sort controls: verify each sort mode changes ordering and persists.
9. `CEF09` Group-by mode: verify grouping by role and site.
10. `CEF10` Saved default tab: set tab, refresh, confirm tab restore.
11. `CEF11` Refresh interval persistence: change interval, refresh, confirm retained.
12. `CEF12` Kiosk mode hotkey/flag: test `k` and `?kiosk=1`.
13. `CEF13` Keyboard shortcut overlay: open with `?` and close with Escape.
14. `CEF14` Legend panel: verify status legend entries render.
15. `CEF15` Status-change highlights: transition device state and confirm highlight class.
16. `CEF16` Stale-data banner: stop polling source/network and confirm stale warning.
17. `CEF17` API degraded banner: force bad source/network and confirm retry/backoff banner.
18. `CEF18` Snapshot fallback: fail API, confirm render uses cached data when present.
19. `CEF19` Browser notifications: enable toggle, grant permission, trigger new offline transition.
20. `CEF20` Soft chime profile: toggle soft/default and verify alert playback volume difference.
21. `CEF21` Theme presets: cycle classic/high-contrast/light and confirm persistence after reload.
22. `CEF22` Font scaling: cycle normal/large/xlarge and confirm persistence after reload.
23. `CEF23` PNG snapshot export: click export and verify timestamped PNG download.
24. `CEF24` Preferences JSON import/export: export, import same file, verify settings restore.
25. `CEF25` What's New modal: open from header and confirm release notes + seen-state behavior.

## Notes

- Run CE checks with CE entitlement and strict CE profile where applicable.
- When testing browser notifications, use live source transitions (not simulation/demo fallback).
