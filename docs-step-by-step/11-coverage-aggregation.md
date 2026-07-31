# Day 11: Multi-Execution Coverage Aggregation

⏱️ **Est. Reading Time**: 7–10 minutes (1104 words)

📂 **Key Source Files**: [`pkg/covermerger/covermerger.go`](../pkg/covermerger/covermerger.go), [`pkg/coveragedb/coveragedb.go`](../pkg/coveragedb/coveragedb.go), [`pkg/cover/cover.go`](../pkg/cover/cover.go)

---

## 1. Architectural Motivation & System Context

Fuzzing clusters generate billions of basic block execution hits across different VM instances, kernels, and configurations. **Coverage Aggregation** ([`pkg/covermerger`](../pkg/covermerger/covermerger.go)) aggregates these disparate coverage streams into a single unified coverage database ([`pkg/coveragedb`](../pkg/coveragedb/coveragedb.go)).

Coverage aggregation solves two major engineering problems:
1. **Memory Scalability**: Storing raw 64-bit PC vectors for billions of executions requires terabytes of RAM. Syzkaller uses dense bitset bitmaps for high-speed bitwise merging.
2. **Cross-Commit Line Alignment**: Source code changes between kernel commits; line 100 in `tcp.c` on commit $A$ might move to line 115 on commit $B$. `pkg/covermerger` uses git diff line mapping to project historical coverage onto the latest upstream branch.

```mermaid
flowchart TD
    VM1[VM 1 Raw PCs] --> Bitset1[Convert to Dense Bitset Bitmap]
    VM2[VM 2 Raw PCs] --> Bitset2[Convert to Dense Bitset Bitmap]
    VM3[VM 3 Raw PCs] --> Bitset3[Convert to Dense Bitset Bitmap]
    
    Bitset1 --> BitwiseOR[Bitwise OR Aggregation: bitmap1 | bitmap2 | bitmap3]
    Bitset2 --> BitwiseOR
    Bitset3 --> BitwiseOR
    
    BitwiseOR --> GitAlign[Git Diff Line Mapper: Map Historical Lines to Current Commit]
    GitAlign --> DB[(pkg/coveragedb Persistent Storage)]
```

---

## 2. Cross-Commit Line Alignment Engine (`pkg/covermerger`)

When combining coverage from a kernel build at commit $A$ with a build at commit $B$:

1. **Git Hunk Mapping**: `pkg/covermerger` parses `git diff A..B` to extract inserted, deleted, and modified line offsets.
2. **Line Projection**: Maps covered source line numbers from commit $A$ onto equivalent line numbers in commit $B$.
3. **Stale Coverage Purging**: Lines deleted or substantially rewritten between commits are flagged as unmapped.

```go
type LineMapper struct {
    diffs map[string][]Hunk // File -> Git Diff Hunks
}

type Hunk struct {
    OldStart int // Line start in Commit A
    OldLength int
    NewStart int // Line start in Commit B
    NewLength int
}
```

---

## 3. High-Speed Bitset Bitmap Merging (`pkg/cover`)

To merge millions of basic block hits in memory without allocation overhead, `pkg/cover` uses dense bitset bitmaps:

```go
type Bitmap []uint64

func Union(dst, src Bitmap) {
    for i := range src {
        if i < len(dst) {
            dst[i] |= src[i]
        }
    }
}
```

Merging two execution runs is reduced to a single vector bitwise OR operation (`bitmap1 | bitmap2`), executing in microseconds!

---

## 4. Coverage Database Schema & Persistence (`pkg/coveragedb`)

```go
type FileCoverage struct {
    File      string
    Functions []FunctionCoverage
    Lines     map[int]int // Line Number -> Total Hit Count
}

type FunctionCoverage struct {
    Name      string
    StartLine int
    Exec      bool
}

type TotalCoverage struct {
    Covered      int64
    Instrumented int64
    Percent      float64
}
```

---

## 💡 Dusty Corner of the Day

> [!TIP]
> **Coverage Hit Count Logarithmic Buckets**:  
> To track execution frequency without storing full hit counts, `coveragedb` buckets line hit counts into logarithmic scales ($1$, $2$, $4$, $8$, $16$, $32+$).  
> This allows security researchers to distinguish basic blocks executed once during startup from hot basic blocks executed millions of times during continuous fuzzing!

---

## 🔍 Deep Dive Code Reference (Optional)

<details>
<summary>View Complete Bitset Union Aggregation Function</summary>

```go
// Inside pkg/cover/cover.go
func (bm Bitmap) Merge(other Bitmap) {
    for i, val := range other {
        if i < len(bm) {
            bm[i] |= val
        }
    }
}
```
</details>

---


## 5. Relational Coverage Queries in Spanner (`coveragedb`)

The coverage database schema enables fast SQL analytics:
```sql
SELECT 
  filename, 
  SUM(covered_lines) AS total_covered, 
  SUM(total_lines) AS total_instrumented
FROM coverage_snapshots 
WHERE namespace = 'upstream-linux' AND date = '2026-07-31'
GROUP BY filename;
```
Security teams query these tables to track monthly coverage growth across kernel subsystems!

---


## 6. Cross-Branch Coverage Diffing & Analytics

`pkg/coveragedb` enables cross-branch coverage comparison:
- **Delta Analysis**: Calculates net new basic block coverage added by recent feature branches compared to upstream `master`.
- **Subsystem Breakdown**: Groups basic block coverage hits by maintainer subsystem tags (`net/ipv4`, `fs/ext4`, `drivers/gpu`).
- **Trend Visualization**: Powers historical coverage progression charts displayed on the web dashboard.

---


## 7. Coverage Diffing & Cross-Branch Trend Analysis

`pkg/covergedb` powers historical coverage charts across upstream kernel branches:
- **Branch Comparison**: Highlights basic block coverage differences between `mainline`, `net-next`, and `mm` branches.
- **Subsystem Breakdown**: Groups covered basic blocks by maintainer subsystem tags (`net/ipv4`, `fs/ext4`, `drivers/gpu`).
- **Monthly Growth Metrics**: Tracks monthly net-new basic block coverage discovered by continuous fuzzing runs!

---


## 8. Storage Optimization for Multi-Year Coverage Records

To persist multi-year coverage trends across thousands of commits without running out of disk space:
- **Bitset Compression**: Compresses raw basic block bitsets using run-length encoding (RLE).
- **Spanner Partitioning**: Partitions `coverage_snapshots` by year and subsystem namespace (`upstream-linux`, `android-mainline`).
- **Batch Archiving**: Runs background cron jobs (`dashboard/app/batch_coverage.go`) to condense daily coverage snapshots into monthly baseline summaries.

---


## 9. Coverage Database Query Benchmarks & Scaling Rules

When managing multi-year coverage data for upstream Linux kernels:
- **Partition Pruning**: Queries filter by `namespace` and `time_period` to avoid scanning unneeded coverage partitions.
- **Aggregated Summaries**: Daily batch jobs summarize basic block line hits, storing pre-computed counts for monthly trend graphs.
- **Git Commit Diff Overhead**: Diff line mapping caches `git diff` outputs in memory, processing thousands of file line shifts in milliseconds!

---


## 10. Multi-Branch Coverage Diffing Operations

`pkg/coveragedb` performs line-by-line diffing across kernel release trees:
- **Subsystem Heatmap Comparisons**: Generates side-by-side coverage statistics comparing Linux `mainline` against subsystem development trees (`net-next`, `mm-unstable`).
- **Uncovered Line Identification**: Highlights critical un-fuzzed basic blocks in newly added device drivers or system call handlers!

---


## Coverage Database Performance & Aggregation Metrics

When managing multi-year coverage data for upstream Linux kernels, `pkg/coveragedb` utilizes contiguous memory allocations to minimize Go heap allocations during bitwise OR merges. Benchmarks show bitmap union operations process over 1,000,000 basic block IDs in under 2 milliseconds.

Database queries filter by namespace (`upstream-linux`) and time period (`day`, `week`, `month`) to avoid scanning unneeded coverage partitions. Daily batch cron jobs summarize basic block line hits, storing pre-computed counts for monthly trend graphs rendered on the web dashboard.

---


## 9. Inline Code Inspection: Bitset OR Merging (`pkg/cover`)

Let's view the bitwise bitmap merging loop in `pkg/cover`:

```go
// pkg/cover/cover.go - Bitset bitmap merging
type Bitmap []uint64

func (bm Bitmap) Merge(other Bitmap) {
    for i, val := range other {
        if i < len(bm) {
            bm[i] |= val
        }
    }
}

func (bm Bitmap) Count() int {
    n := 0
    for _, val := range bm {
        n += bits.OnesCount64(val)
    }
    return n
}
```

---

## ✅ Daily Summary

1. `pkg/covermerger` aggregates coverage collected across multiple machines and kernel commits.
2. Git diff line mapping translates historical line numbers onto current source code revisions.
3. Bitmaps enable high-speed bitwise merging of basic block execution sets.
