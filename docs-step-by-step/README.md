# 📚 Syzkaller Implementation Step-by-Step: Vacation Study Guide

Welcome to your 21-day vacation study guide for **syzkaller**! Designed for 15–20 minutes of daily reading, this guide focuses on syzkaller's major subsystems: `syz-manager`, coverage processing, `syz-ci`, and the web `dashboard`.

---

## 📱 Mobile Reading Options

### Option A: Read directly on GitHub (Mobile App / Web)
You can read these `.md` files directly on GitHub. Every file includes native code syntax highlighting and Mermaid architecture diagrams.

### Option B: Deploy to GitHub Pages (Recommended for Mobile)
For an e-reader experience with bottom-of-the-page **"Next / Previous Chapter"** buttons and responsive text formatting:
1. Go to your repository settings on GitHub: **Settings -> Pages**.
2. Set the Source branch to `main` (or `master`) and directory to `/docs-step-by-step`.
3. GitHub will generate a clean web app at `https://<your-username>.github.io/<repo-name>/`!

---

## 🗓️ 21-Day Curriculum Overview

```
       ┌─────────────────────────────────────────────────────────┐
       │             Syzkaller Architecture Mastery              │
       └────────────────────────────┬────────────────────────────┘
                                    │
    ┌───────────────────┬───────────┴───────┬───────────────────┐
    ▼                   ▼                   ▼                   ▼
Module 1: syz-mgr   Module 2: Coverage  Module 3: syz-ci    Module 4: Dashboard
(Days 1-7)          (Days 8-12)         (Days 13-16)        (Days 17-21)
```

### **Module 1: `syz-manager` & Core Orchestration**
- [ ] [Day 01: `syz-manager` Overview & Startup Lifecycle](01-syz-manager-overview.md) — *(9–12 min, 1453 words)*
- [ ] [Day 02: VM Management & Instance Abstractions](02-vm-management.md) — *(8–11 min, 1266 words)*
- [ ] [Day 03: RPC Protocol & Fuzzer Communication](03-rpc-protocol.md) — *(7–10 min, 1188 words)*
- [ ] [Day 04: Corpus Management & Signal Operations](04-corpus-and-signal.md) — *(7–10 min, 1162 words)*
- [ ] [Day 05: Crash Parsing & Deduplication Engine](05-crash-processing.md) — *(7–10 min, 1119 words)*
- [ ] [Day 06: Automated Reproducer Pipeline](06-reproducer-pipeline.md) — *(7–10 min, 1106 words)*
- [ ] [Day 07: Syz-Hub & Inter-Manager Corpus Sharing](07-syz-hub.md) — *(7–10 min, 1128 words)*

### **Module 2: Kernel Coverage Engine & Processing**
- [ ] [Day 08: Kernel Coverage Fundamentals (KCOV & Sanitizers)](08-kcov-and-sanitizers.md) — *(7–10 min, 1132 words)*
- [ ] [Day 09: PC Processing & Signal Transformation](09-pc-processing-and-signal.md) — *(7–10 min, 1094 words)*
- [ ] [Day 10: Symbolization & Address Resolution](10-symbolization.md) — *(7–10 min, 1126 words)*
- [ ] [Day 11: Multi-Execution Coverage Aggregation](11-coverage-aggregation.md) — *(7–10 min, 1104 words)*
- [ ] [Day 12: Coverage Visualization & HTML Generation](12-coverage-visualization.md) — *(7–10 min, 1144 words)*

### **Module 3: Continuous Integration with `syz-ci` & Bisection**
- [ ] [Day 13: `syz-ci` Architecture & Manager Supervision](13-syz-ci-architecture.md) — *(7–10 min, 1082 words)*
- [ ] [Day 14: VCS Automation & Kernel Build Pipeline](14-vcs-and-build-pipeline.md) — *(7–10 min, 1063 words)*
- [ ] [Day 15: Automated Bug Bisection Engine](15-bisection-engine.md) — *(7–10 min, 1128 words)*
- [ ] [Day 16: CI Task Scheduler & Resource Isolation](16-ci-task-scheduler.md) — *(6–9 min, 1030 words)*

### **Module 4: Syzbot Dashboard & Bug Lifecycle**
- [ ] [Day 17: Dashboard Architecture & Datastore Schema](17-dashboard-architecture.md) — *(7–10 min, 1123 words)*
- [ ] [Day 18: `dashapi` Protocol & Client Interop](18-dashapi-protocol.md) — *(6–9 min, 1025 words)*
- [ ] [Day 19: Bug Lifecycle & Email Reporting Engine](19-bug-lifecycle-and-email.md) — *(6–9 min, 1006 words)*
- [ ] [Day 20: External Patch Testing (`#syz test`)](20-patch-testing.md) — *(7–10 min, 1108 words)*
- [ ] [Day 21: Subsystem Mapping & Automated Triage](21-subsystem-triage-and-ai.md) — *(6–9 min, 1048 words)*

---

## 🎨 Visual Chapter Layout

Each chapter uses a consistent bite-sized layout:
1. **Header Metadata**: Estimated reading time & core source file links.
2. **Architecture Diagram**: Mermaid diagram for immediate visual grounding.
3. **Core Concepts**: Clear explanations of structs, APIs, and data flows.
4. **💡 Dusty Corner of the Day**: Edge-case highlights and subtle implementation tricks.
5. **✅ Daily Summary**: 3 quick takeaways.
