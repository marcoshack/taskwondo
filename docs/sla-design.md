# TF-69: Project SLA Definitions

## Context

Project owners/admins need to define Service Level Agreements (SLAs) per workflow status, scoped by work item type. For example: "Bug items in status Open must transition out within 24h." SLAs track how long an item stays in each status, with anti-gaming via accumulated elapsed time (status round-trips don't reset the timer). For now, SLAs are display-only (remaining time + color indicators) — notifications/alerts come later.

This also drops the unused `sla_deadline` column (redundant with `due_date`, never used).

---

## Step 1: Migration `000017_create_sla_tables`

**New file:** `api/internal/database/migrations/000017_create_sla_tables.up.sql`

```sql
-- SLA targets: max time per status, scoped by project + type + workflow
CREATE TABLE sla_status_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    work_item_type  TEXT NOT NULL,
    workflow_id     UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status_name     TEXT NOT NULL,
    target_seconds  INT NOT NULL CHECK (target_seconds > 0),
    calendar_mode   TEXT NOT NULL DEFAULT '24x7' CHECK (calendar_mode IN ('24x7', 'business_hours')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, work_item_type, workflow_id, status_name)
);
CREATE INDEX idx_sla_status_targets_project ON sla_status_targets(project_id);

-- Elapsed time per work item per status (anti-gaming accumulation)
CREATE TABLE work_item_sla_elapsed (
    work_item_id    UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    status_name     TEXT NOT NULL,
    elapsed_seconds INT NOT NULL DEFAULT 0,
    last_entered_at TIMESTAMPTZ,
    PRIMARY KEY (work_item_id, status_name)
);

-- Business hours config on projects
ALTER TABLE projects ADD COLUMN business_hours JSONB;

-- Drop unused sla_deadline (redundant with due_date)
ALTER TABLE work_items DROP COLUMN sla_deadline;
```

**New file:** `000017_create_sla_tables.down.sql` (reverse all changes)

---

## Step 2: Model layer

**New file:** `api/internal/model/sla.go`
- `SLAStatusTarget` struct (ID, ProjectID, WorkItemType, WorkflowID, StatusName, TargetSeconds, CalendarMode, timestamps)
- `SLAElapsed` struct (WorkItemID, StatusName, ElapsedSeconds, LastEnteredAt)
- `SLAInfo` struct (computed: TargetSeconds, ElapsedSeconds, RemainingSeconds, Percentage, Status)
- `BusinessHoursConfig` struct (Days []int, StartHour, EndHour int, Timezone string)
- Constants: `CalendarMode24x7`, `CalendarModeBusinessHours`, `SLAStatusOnTrack/Warning/Breached`

**Modify:** `api/internal/model/workitem.go` — Remove `SLADeadline *time.Time` (line 55)

**Modify:** `api/internal/model/project.go` — Add `BusinessHours *BusinessHoursConfig` to Project struct

---

## Step 3: Repository layer

**New file:** `api/internal/repository/sla.go` — follows `repository/project_type_workflow.go` pattern
- `ListTargetsByProject(ctx, projectID)` — all targets for project
- `ListTargetsByProjectAndType(ctx, projectID, type, workflowID)` — targets for a type+workflow
- `BulkUpsertTargets(ctx, targets [])` — INSERT ON CONFLICT DO UPDATE
- `DeleteTarget(ctx, id)`
- `InitElapsedOnCreate(ctx, workItemID, statusName, enteredAt)` — INSERT initial elapsed record
- `UpsertElapsedOnEnter(ctx, workItemID, statusName, now)` — upsert with last_entered_at = now
- `UpdateElapsedOnLeave(ctx, workItemID, statusName, now)` — `elapsed_seconds += EXTRACT(EPOCH FROM (now - last_entered_at))::INT`, set `last_entered_at = NULL`
- `GetElapsed(ctx, workItemID, statusName)` — single lookup
- `ListElapsedByWorkItemIDs(ctx, ids []uuid.UUID)` — batch load for list views (avoid N+1)

**Modify:** `api/internal/repository/workitem.go` — Remove `sla_deadline` from all SQL (SELECT, INSERT, UPDATE, scan functions). ~4 locations.

**Modify:** `api/internal/repository/project.go` — Add `business_hours` JSONB to SELECT/INSERT/UPDATE, unmarshal in scan.

---

## Step 4: Service layer

**New file:** `api/internal/service/sla.go`
- `SLAService` struct with dependencies: SLARepository, ProjectRepository, ProjectMemberRepository, WorkflowRepository
- `ListTargets(ctx, userInfo, projectKey)` — auth: member can read
- `BulkUpsertTargets(ctx, userInfo, projectKey, input)` — auth: owner/admin. Validates: statuses exist in workflow, rejects terminal statuses (done/cancelled category)
- `DeleteTarget(ctx, userInfo, projectKey, targetID)` — auth: owner/admin
- `ComputeSLAInfo(ctx, workItem, projectID, workflowID)` — look up target + elapsed, compute remaining/percentage/status. Returns nil if no SLA target. For business_hours mode, uses `CalculateBusinessSeconds`.
- `CalculateBusinessSeconds(from, to time.Time, config BusinessHoursConfig) int` — iterates day-by-day, only counting seconds within configured business hours/days

**Modify:** `api/internal/service/workitem.go`
- Add `slaRepo` dependency to `WorkItemService` struct and constructor
- **On Create:** after item is created, call `slaRepo.InitElapsedOnCreate(ctx, item.ID, item.Status, now)`
- **On status change** (lines ~440-473): after existing status change logic, add:
  - `slaRepo.UpdateElapsedOnLeave(ctx, item.ID, oldStatus, now)` — close out old status
  - `slaRepo.UpsertElapsedOnEnter(ctx, item.ID, newStatus, now)` — start tracking new status
- **On Get/List:** enrich work item responses with computed SLA info (batch for list)

---

## Step 5: Handler layer + routes

**New file:** `api/internal/handler/sla.go` — follows `handler/milestone.go` pattern
- Request DTOs: `bulkUpsertSLARequest` (work_item_type, workflow_id, targets[])
- Response DTO: `slaTargetResponse`
- Handlers: `List` (GET), `BulkUpsert` (PUT), `Delete` (DELETE)

**Modify:** `api/internal/handler/workitem.go`
- Remove `SLADeadline` from `workItemResponse` and `toWorkItemResponse`
- Add `SLA *slaInfoResponse` to response
- Add `sla_status` filter param support (on_track/warning/breached)

**Modify:** `api/internal/handler/project.go` — Add `business_hours` to project DTOs

**Routes** in `api/cmd/server/main.go`:
```
/api/v1/projects/{projectKey}/sla-targets     GET, PUT
/api/v1/projects/{projectKey}/sla-targets/{id} DELETE
```

Wire: `slaRepo`, `slaService`, `slaHandler` in main.go (line ~107+)

---

## Step 6: Backend tests

**New file:** `api/internal/service/sla_test.go` (~12 tests)
- BulkUpsert: auth checks, terminal status rejection, happy path
- ComputeSLAInfo: on_track / warning / breached calculations
- CalculateBusinessSeconds: various date ranges, weekends, partial days
- Elapsed accumulation: enter → leave → re-enter round-trip

**New file:** `api/internal/handler/sla_test.go` (~6 tests)
- List, BulkUpsert, Delete endpoints
- Auth enforcement (member reads, admin writes)

---

## Step 7: Frontend API + hooks + utilities

**New file:** `web/src/api/sla.ts` — SLATarget, SLAInfo, BulkUpsertSLAInput types + API functions

**New file:** `web/src/hooks/useSLA.ts` — `useSLATargets`, `useBulkUpsertSLATargets`, `useDeleteSLATarget`

**New file:** `web/src/utils/duration.ts`
- `parseDuration("1d 2h")` → seconds (or null if invalid)
- `formatDuration(seconds)` → "1d 2h"
- `formatRemaining(seconds)` → "2h 15m left" / "1h 30m overdue"
- Supports: m (minutes), h (hours), d (days), w (weeks)

**Modify:** `web/src/api/workitems.ts` — Remove `sla_deadline`, add `sla: SLAInfo | null`

---

## Step 8: SLA config modal + project settings

**New file:** `web/src/components/SLAConfigModal.tsx`
- Opened from clock icon in project settings workflows section
- Shows all statuses for the selected workflow
- Duration input per non-terminal status (text input, parsed with `parseDuration`)
- Calendar mode dropdown per status (24/7 or Business Hours)
- Terminal statuses (done/cancelled) shown disabled with tooltip explaining why
- Save via `bulkUpsertSLATargets` mutation + green checkmark feedback

**Modify:** `web/src/pages/ProjectSettingsPage.tsx`
- Add `Clock` icon button next to each type's workflow select (line ~395)
- State for SLA modal: `slaModalType` + `slaModalWorkflowId`
- Render `<SLAConfigModal>` when state is set

**Add Business Hours section** to ProjectSettingsPage (below Workflows, above Complexity):
- Day checkboxes (Mon-Sun), start/end hour selects, timezone select
- Saved via project update mutation with `business_hours` field

---

## Step 9: SLA display components

**New file:** `web/src/components/SLAIndicator.tsx`
- Props: `sla: SLAInfo | null`, `compact?: boolean`
- Color: green (<75%), yellow (75-99%), red (>=100%)
- Shows: clock icon + formatted remaining time
- Renders nothing if `sla` is null

**Modify:** `web/src/pages/WorkItemListPage.tsx` — Add SLA column to DataTable
**Modify:** `web/src/pages/WorkItemDetailPage.tsx` — Add SLA indicator in sidebar
**Modify:** `web/src/components/workitems/BoardView.tsx` — Add compact SLA indicator on cards
**Modify:** `web/src/components/workitems/WorkItemFilters.tsx` — Add SLA Status filter dropdown

---

## Step 10: i18n

**Modify:** `web/src/i18n/en.json` — ~25 new keys for SLA labels, tooltips, filter options, durations, business hours config

---

## SLA Elapsed Time Logic (Anti-Gaming)

```
Enter status X:  UPSERT elapsed(work_item_id, X) SET last_entered_at = now()
Leave status X:  UPDATE elapsed SET elapsed_seconds += (now - last_entered_at), last_entered_at = NULL
Return to X:     elapsed_seconds carries forward (no reset)

Remaining = target_seconds - elapsed_seconds - (now - last_entered_at)
Breached = remaining < 0
```

---

## Key Design Decisions

1. **SLA targets are per (project, type, workflow, status)** — changing workflow preserves old targets
2. **Calendar mode is per SLA target** — allows mixed modes within a project/type
3. **Elapsed tracking via `work_item_sla_elapsed` table** — separate from work_items, future-proof
4. **Business hours configurable per project** — stored as JSONB on projects table
5. **SLA filtering (breached)** — post-filter in service layer for V1, can add materialized column later
6. **`sla_deadline` dropped** — unused, redundant with `due_date`

---

## Verification

1. Run `make test` — all existing tests pass after sla_deadline removal
2. Run migration against local DB — tables created, column dropped
3. Create SLA targets via API — bulk upsert, verify persistence
4. Create work item, change status — verify elapsed tracking
5. Status round-trip (A→B→A) — verify elapsed accumulates
6. Frontend: configure SLAs in project settings — modal works, saves
7. Frontend: verify SLA indicators in list, detail, board views
8. Frontend: filter by breached — returns correct items
9. Business hours: verify countdown only ticks during configured hours
10. `npm run build` — frontend builds clean
