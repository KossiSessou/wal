# <Component> On-Disk Format

**Status:** draft
**Last updated:** 2026-07-02

## Purpose
This format describes a WAL record so that readers with no external background can tell how far to read from each record via the length field and check the integrity of the data(payload field) by calculating the checksum.

## Record Layout

| Offset | Field        | Width    | Type            | Description                  |
|--------|--------------|----------|-----------------|------------------------------|
| 0      | Length       | 4 bytes  | uint32 LE       | Length of the payload        |
| 4      | Checksum     | 4 bytes  | uint32 LE       | Checksum of Length + Payload |
| 8      | Payload      | N bytes  | variable        | Payload                      |

Header = 8 bytes fixed. Total = 8 + N.

## Field Decisions

- **Length (offset 0):** This tells the reader how many bytes to read for the payload
- **Checksum before payload:** This is ideal for our design as we have our entire payload at hand when we call append. A checksum after payload is ideal for streaming which is not our case. We can also easily access the length and checksum easily as we have 8 bytes fixed size header.
- **Checksum Length + Payload(Offset 4):** reader must know whether the length or the payload has been corrupted. A corrupted length is dangerous as the reader might try to access invalid memory address that could cause the program to issue a seg fault or access garbage data.
- **Little-endian:** matches the host architecture (x86/ARM); reader
  and writer must agree, and this is the project-wide convention.

## Segment Formant and Naming
A segment is a file of record appended back to back.
```txt
0000000001.wal:  [record A][record B][record C]...[record M]
                ↑0       ↑89       ↑184
                (byte positions inside THIS file)

00000000002.wal:  [record N][record O][record P]...
                ↑0       ↑42

0000000003.wal:  [record X][record Y]...
                ↑0       ↑1247  ← Offset{SegmentID: 3, Position: 1247} points here
```
Each follow the form `%010d.wal`. Zero-padded 10 digit.  
  
```txt
/data/mywal/
    0000000001.wal     ← oldest, immutable (frozen, no longer being written)  
    0000000002.wal     ← immutable  
    0000000003.wal     ← immutable  
    0000000004.wal     ← ACTIVE — currently being appended to  
```

## The Offset Model
The offset is a `struct {SegmentID uint64, Position uint64}`. The SegementID field refers to the current segment in use an the Postion field refers to the byte offset of the record first byte within that file.

## Segment Rotation Rule
A segment is 64MB large. If a to be appended record size exceed it, it will be rejected with `ErrRecordTooLarge`. If appending a record will make the existing file exceed that capacity, we trigger a rotation. Once rotated away, segments are never written to again.

## `Wal.Open(dir, cfg)` Logic
The reader calls `Open` with a directory path and a `Config` settings. If the directory does not exist we create one. We then pull all the entries from it. If there is no entry, we set WAL.activeID to 1, otherwise we pick the segment with the highest filename and use it as our WAL.activeID. We then open a file with the equivalent name and use it as our `WAL.fd`


## Invariants
Things that are always true if the file is valid:
- Every record begins with Length.
- A reader at any record boundary can compute the next boundary as
  current_offset + 8 + N. 
- No segment exceeds MaxSegmentSize

## Decision Log
Decision 1 — zero-padded sequence numbers (e.g., 00000001.wal). Two practical wins: it sorts lexicographically the same as numerically (so ls shows them in order), and operational tools (ls, grep, glob patterns) handle them cleanly.
Decision 2 — rotate before writing (close the active segment when the next record would exceed MaxSegmentSize). It makes MaxSegmentSize a hard upper bound that's never violated, instead of a soft target. Replay code can now assume "no segment exceeds MaxSegmentSize" as an invariant. 
## Decisions Deferred
What this format intentionally does NOT handle, and why:
- **No versioning:** format is frozen for this project; revisit if it
  must evolve.
- **No compression:** out of scope.

## Open Questions
Things not yet resolved:
- How is the end-of-file / unused tail distinguished from a real record?