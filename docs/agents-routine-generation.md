# Agents: Professional Routine Generation

The `agents` package generates daily routines (time-blocked schedules) for a
broad catalog of professions. Each profession is backed by its own **agent**, so
routines can be produced for a single professional or for **all professionals**
in one call.

## Concepts

| Type | Role |
| --- | --- |
| `Profession` | A discipline (physician, software engineer, electrician, …) with work hours and core activities. |
| `Activity` | A reusable unit of work an agent lays into a routine. |
| `Agent` | A profession-bound routine generator (`NewAgent(profession)`). |
| `Registry` | The fleet of agents; `DefaultRegistry()` covers the full built-in catalog. |
| `Routine` | A generated day: ordered, non-overlapping `TimeBlock`s. |
| `Service` | Wires the registry to a `RoutineStore` and an `EventPublisher`. |

## Generating routines

```go
reg := agents.DefaultRegistry()

// One professional.
routine, _ := reg.Generate("software-engineer", agents.GenerateOptions{
    Intensity:       agents.IntensityBalanced,
    IncludeWellness: true,
    IncludeBreaks:   true,
})

// Every professional in the catalog.
all, _ := reg.GenerateAll(agents.GenerateOptions{})
```

`GenerateOptions` tunes the output:

- **`Intensity`** — `light`, `balanced` (default), or `intense`; scales activity
  durations.
- **`IncludeWellness`** — prepends a morning exercise/planning block and a
  wind-down block.
- **`IncludeBreaks`** — inserts short recovery breaks between demanding blocks.
- **`FocusAreas`** — extra focus blocks appended for the day.
- **`Date`** — the day the routine is for (defaults to today).

A midday meal is inserted automatically around the midpoint of the work day, and
overnight/24-hour shifts (e.g. firefighter, pilot) wrap correctly past midnight.

## Service, persistence, and events

`Service` persists each routine through a `RoutineStore` and announces it:

- `GenerateRoutine(slug, opts)` → `EventRoutineGenerated`
- `GenerateForAll(opts)` → one `EventBatchGenerated` (results sorted by slug)

## HTTP API

`Handler.RegisterRoutes` mounts:

| Method & path | Description |
| --- | --- |
| `GET /api/v1/agents` | List the profession catalog. |
| `POST /api/v1/agents/{slug}/routines` | Generate a routine for one profession. |
| `POST /api/v1/routines/generate-all` | Generate routines for all professionals. |

Request bodies are `GenerateOptions` JSON; an empty body uses defaults.
