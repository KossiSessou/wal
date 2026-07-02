# Runtime Behavior

**status:** draft
**last updated:** 2026-07-02

## Sync Policies
The WAL implement three policies:
`SyncAlways`: every successful `Append` return means the record survives any crash. Slowest, strongest.
`SyncInterval(d)`: after a crash, records appended in the last ~d may be lost. Everything older survives.
`SyncNever`: after a crash, an unbounded amount of trailing data may be lost. The OS flushes on its own schedule. Fastest, weakest.

There is also a manual `Sync()` that lets a caller force durability on demand in any mode. 
`Close()` performs a best-effort sync before closing (which strengthens Close's contract to "when Close returns, everything appended so far is durable").

## Concurrency Contract
`Append` and `Sync` are safe to call concurrently from multiple goroutines; the WAL serializes them internally via mutex. `Close` is safe to call multiple times (via sync.Once) but must be called at most once from a lifecycle perspective. Callers holding a returned `Offset` across a `Close` and `Open` are fine, offsets are stable identities on disk.

## Lifecycle

A WAL instance moves through three states:
```txt
                              Open(dir, cfg)             Close()
[Unopened] ─────────────────▶ [Open/running] ──────────▶ [Closed]
no fd,                        fd held,                   fd closed,
no goroutine                  mutex live                 ErrClosed on all ops
```
#### Open (running)

While open, three operations coexist, all serialized by the internal mutex:

| Operation          | Who drives it                | What it does                                    |
|--------------------|------------------------------|-------------------------------------------------|
| `Append(record)`   | caller                       | frames + writes a record; rotates segment if full |
| `Sync()`           | caller                       | forces fsync of the active segment              |
| `syncLoop`         | internal goroutine           | periodic fsync; only exists in `SyncInterval` mode |

#### Close sequence

`Close()` is idempotent (`sync.Once`) and performs, in order:

1. Mark closed under the lock; best-effort `fd.Sync()` so buffered
   appends are durable before shutdown.
2. `close(done)` signals the syncLoop goroutine to exit.
3. If in `SyncInterval` mode, block on the `stopped` rendezvous until
   the goroutine has fully exited (prevents fsync racing fd close).
4. `fd.Close()`.

Once closed, every public method returns `ErrClosed`. There is no reopen
on the same instance. Call `Open` again for a fresh one.

### Invariants

- The state machine is one-way: Unopened → Open → Closed. No transitions backward.
- The syncLoop goroutine's lifetime is strictly contained within the Open state —
  it is spawned by `Open` and provably exited before `Close` returns.
- After `Close` returns nil, everything ever appended is durable (the step-1 sync
  strengthens Close's contract beyond just resource cleanup).

## Decision Log

Decision 1 — (segment_id, position) tuple over Global offset. Replay is trivial: open segment_id, seek to position. The Phase 1 index stores it as a struct, slightly larger per entry, but unambiguous. Rotation doesn't require any global accounting; each segment is internally self-consistent.