## 2026-11-05 - Auth middleware double-lookup anti-pattern
**Learning:** `Server.Authenticate` called `LookupAPIKey()` then `ValidateAPIKey()`, and `ValidateAPIKey` re-ran `LookupAPIKey` internally — every authenticated request paid 2 identical SELECTs. Composite store methods that "check + record" hide a second query when the caller already holds the record.
**Action:** When a store has lookup+mutate pairs, expose a record-level mutator (`RecordAPIKeyUsage(rec)`) and audit call sites for validate-after-lookup duplication. Check `ValidateToken` callers too — it still uses the combined path where only a bool is needed.
## 2026-11-05 - gopsutil cpu.Percent(0) is unsafe with mixed callers
**Learning:** `cpu.Percent(0)` computes delta since the *previous* gopsutil call, and that state is shared process-wide. The 5s background aggregator and on-demand stats handlers interleave, so interval=0 readings become jittery/unattributable. Rejected switching `GetDetailedStats` to non-blocking percent — the 200ms block is bounded and correct.
**Action:** For CPU metrics here, keep an explicit interval or source the value from the aggregator's last sample instead of calling gopsutil ad-hoc.

## 2026-11-05 - Catalog endpoint was a filesystem storm
**Learning:** `GetPluginCatalog` ran `findPluginDeb`/`findPluginChangelog` per entry — hundreds of glob/stat probes per request, polled every 12s. Per-request "assemble from filesystem" handlers need TTL caching keyed to mutation points (install/uninstall/refresh), not just fast code.
**Action:** When a polled endpoint aggregates file discovery, cache the assembled response and invalidate at every mutation site. `Manager.Discover()` now exposes `GetDiscoveredManifests()` snapshot for read paths.

## 2026-11-05 - gopsutil MemoryPercent double-reads /proc
**Learning:** `p.MemoryPercent()` internally calls `mem.VirtualMemory()` AND `p.MemoryInfo()` — calling both it and `MemoryInfo()` costs 2 statm reads + N VirtualMemory reads per process. Deriving percent from one `MemoryInfo` + one `VirtualMemory` is identical math (RSS/total*100).
**Action:** Check gopsutil source before assuming a convenience method is cheap — several compose other calls.
