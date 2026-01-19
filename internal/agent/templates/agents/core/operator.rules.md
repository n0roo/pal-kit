# Operator Agent Rules

> Claude 참조용 운영/연속성 관리 규칙

---

## Quick Reference

```
Type: Core Agent
Role: Operations & Continuity Management
Integrates: Manager + Logger roles
Triggers: SessionStart, SessionEnd, PortComplete
```

---

## Directory Structure

```
.pal/
├── sessions/                # Session records
│   └── {date}-{id}.md
├── decisions/               # ADRs
│   └── ADR-{num}-{title}.md
└── context/                 # Context files
    ├── current-state.md
    ├── session-briefing.md
    └── dashboard.md
```

---

## Session Start Protocol

### Steps
```
1. Load previous session → .pal/sessions/
2. Check current state → pal status
3. Identify active ports & blockers
4. Output briefing
```

### Briefing Format
```markdown
## Session Briefing

### Last Session Summary
- Date: {date}
- Completed: {tasks}
- In Progress: {tasks}

### Current State
- Active Ports: {list}
- Blockers: {list or "None"}

### Recommended Next
1. {priority task}
2. {next task}
```

---

## Session End Protocol

### Steps
```
1. Collect session work
2. Identify decisions (ADR candidates)
3. Generate session summary
4. Save to .pal/sessions/{date}-{id}.md
5. Record next priorities
```

### Summary Format
```markdown
## Session Summary: {session-id}

### Completed
- {task}: {result}

### Decisions
- {decision}: {rationale}

### Incomplete/Next
- [ ] {next task}

### Notes
- {notes}
```

---

## ADR Management

### When to Create ADR
| Situation | Example |
|-----------|---------|
| Architecture change | Layer structure, new module |
| Tech stack choice | Library, framework |
| Design pattern | CQS, Event Sourcing |
| Trade-off decision | Performance vs maintainability |

### ADR Template
```markdown
# ADR-{num}: {title}

## Status
Proposed | Accepted | Deprecated | Superseded

## Date
{YYYY-MM-DD}

## Context
{background}

## Decision
{choice and reasoning}

## Consequences
- Positive: {benefits}
- Negative: {drawbacks}

## Alternatives Considered
- {alt 1}: {why not}
```

---

## Dashboard Output

```markdown
## Project Dashboard

### Port Status
| Status | Count | Ports |
|--------|-------|-------|
| ✅ Complete | {n} | {list} |
| 🔄 In Progress | {n} | {list} |
| ⏳ Pending | {n} | {list} |

### Progress
▓▓▓▓▓▓▓▓░░ {n}%

### Recent Activity
- {timestamp}: {activity}

### Blockers
- {blocker or "None"}

### Next Priority
1. {task 1}
2. {task 2}
```

---

## Quality Gate (from Manager)

### Check Items
| Item | Method | Required |
|------|--------|----------|
| TC Checklist | Manual | ✅ |
| Build | `pal build` | ✅ |
| Test | `pal test` | ✅ |
| Convention | Linter | ⚠️ |

### Gate Result Format
```markdown
## Quality Gate: {port-id}

### Checklist: ✅ / ⚠️ / ❌
### Build: ✅ / ❌
### Test: ✅ / ⚠️ (n/m passed)

### Result: Pass / Rework Needed
**Reason**: {if rework}
```

---

## Blocker Management

### Blocker Format
```markdown
| ID | Port | Description | Date | Status |
|----|------|-------------|------|--------|
| BLK-001 | {port} | {desc} | {date} | Pending |
```

### Escalation Rules
| Situation | Target | Action |
|-----------|--------|--------|
| Scope exceeded | Builder | New port |
| Architecture Q | Architect | Review |
| Tech decision | Architect/User | Decision |
| Long blocker (3d+) | User | Solution proposal |

---

## PAL Commands

```bash
# Status
pal status
pal status --dashboard

# Session
pal session list
pal session show <id>
pal session summary

# Port
pal port list
pal port status <id>

# Quality Gate
pal gate run <port-id>

# Escalation
pal escalation list
```

---

## Checklist

### Session Start
- [ ] Load previous session
- [ ] Check `pal status`
- [ ] Identify active ports
- [ ] Check blockers
- [ ] Output briefing

### Session End
- [ ] Summarize work
- [ ] Record decisions
- [ ] Create ADR (if needed)
- [ ] Save session record
- [ ] Note next priorities

### Port Complete
- [ ] Run quality gate
- [ ] Record artifacts
- [ ] Update port status
- [ ] Check dependent ports

---

## Integration Points

### Hooks
- `SessionStart` → Load briefing
- `SessionEnd` → Save summary
- `PreCompact` → Preserve context
- `Stop` → Finalize records

### Files Written
- `.pal/sessions/{date}-{id}.md`
- `.pal/decisions/ADR-*.md`
- `.pal/context/current-state.md`
- `.pal/context/session-briefing.md`

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| Project direction change | User | Approval request |
| Long blocker (3d+) | User | Solution proposal |
| Architecture decision | Architect | Review request |
| Schedule delay | User | Priority adjustment |
| Repeated gate failure | User | Status report |

---

<!-- pal:rules:core:operator -->
