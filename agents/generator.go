package agents

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const timeLayout = "15:04"

// clock walks minutes across a single day, formatting positions as "HH:MM".
type clock struct {
	minutes int
}

func newClock(hhmm string) (clock, error) {
	t, err := time.Parse(timeLayout, hhmm)
	if err != nil {
		return clock{}, fmt.Errorf("invalid time %q: %w", hhmm, err)
	}
	return clock{minutes: t.Hour()*60 + t.Minute()}, nil
}

func (c clock) String() string {
	m := ((c.minutes % 1440) + 1440) % 1440
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

func (c clock) add(d time.Duration) clock {
	return clock{minutes: c.minutes + int(d.Minutes())}
}

func (c clock) before(other clock) bool { return c.minutes < other.minutes }

// build assembles the ordered blocks for a profession given the options.
func buildBlocks(p Profession, opts GenerateOptions) ([]TimeBlock, error) {
	start, err := newClock(p.WorkStart)
	if err != nil {
		return nil, err
	}
	end, err := newClock(p.WorkEnd)
	if err != nil {
		return nil, err
	}
	// Shifts that wrap past midnight (e.g. 24-hour firefighter/pilot rotations)
	// are represented by rolling the end past the start.
	if !start.before(end) {
		end = clock{minutes: end.minutes + 1440}
	}

	scale := opts.Intensity.scale()
	var blocks []TimeBlock
	c := start

	if opts.IncludeWellness {
		pre := start.add(-90 * time.Minute)
		blocks = append(blocks,
			block(pre, 45*time.Minute, "Morning exercise & mobility", BlockWellness, PriorityMedium, "Kick-start the day before work."),
			block(pre.add(45*time.Minute), 45*time.Minute, "Breakfast & day planning", BlockPersonal, PriorityMedium, ""),
		)
	}

	blocks = append(blocks, block(c, 30*time.Minute, "Day kickoff & priorities", BlockAdmin, PriorityHigh, "Review inbox, set the top three outcomes."))
	c = c.add(30 * time.Minute)

	lunchAt := midpoint(start, end)
	lunchInserted := false
	sinceBreak := 0

	for i, a := range p.CoreActivities {
		dur := scaleDuration(a.Duration, scale)

		// Insert a meal roughly at the midpoint of the working day.
		if !lunchInserted && !c.before(lunchAt) {
			blocks = append(blocks, block(c, 45*time.Minute, "Lunch break", BlockMeal, PriorityMedium, ""))
			c = c.add(45 * time.Minute)
			lunchInserted = true
			sinceBreak = 0
		}

		blocks = append(blocks, block(c, dur, a.Name, a.Category, a.Priority, ""))
		c = c.add(dur)
		sinceBreak++

		// A short recovery break after every couple of demanding blocks.
		if opts.IncludeBreaks && sinceBreak >= 2 && i < len(p.CoreActivities)-1 {
			blocks = append(blocks, block(c, 15*time.Minute, "Short break", BlockBreak, PriorityLow, "Step away, hydrate, reset."))
			c = c.add(15 * time.Minute)
			sinceBreak = 0
		}
	}

	for _, focus := range opts.FocusAreas {
		focus = strings.TrimSpace(focus)
		if focus == "" {
			continue
		}
		blocks = append(blocks, block(c, scaleDuration(45*time.Minute, scale), "Focus: "+focus, BlockDeepWork, PriorityHigh, "Extra focus area requested for today."))
		c = c.add(scaleDuration(45*time.Minute, scale))
	}

	blocks = append(blocks, block(c, 30*time.Minute, "End-of-day review & handover", BlockAdmin, PriorityMedium, "Log progress, tee up tomorrow."))

	if opts.IncludeWellness {
		c = c.add(30 * time.Minute)
		blocks = append(blocks, block(c, 60*time.Minute, "Wind-down & personal time", BlockPersonal, PriorityLow, ""))
	}

	return blocks, nil
}

func block(start clock, dur time.Duration, title string, cat BlockCategory, prio Priority, notes string) TimeBlock {
	return TimeBlock{
		Start:    start.String(),
		End:      start.add(dur).String(),
		Title:    title,
		Category: cat,
		Priority: prio,
		Notes:    notes,
	}
}

func midpoint(a, b clock) clock {
	return clock{minutes: (a.minutes + b.minutes) / 2}
}

func scaleDuration(d time.Duration, scale float64) time.Duration {
	m := int(float64(d.Minutes())*scale + 0.5)
	// Round to the nearest 5 minutes and keep a sane floor.
	m = ((m + 2) / 5) * 5
	if m < 15 {
		m = 15
	}
	return time.Duration(m) * time.Minute
}

func totalMinutes(blocks []TimeBlock) int {
	total := 0
	for _, b := range blocks {
		s, err1 := newClock(b.Start)
		e, err2 := newClock(b.End)
		if err1 != nil || err2 != nil {
			continue
		}
		d := e.minutes - s.minutes
		if d < 0 {
			d += 1440
		}
		total += d
	}
	return total
}

// generate is the low-level routine builder shared by Agent.Generate.
func generate(p Profession, opts GenerateOptions) (*Routine, error) {
	if p.Slug == "" {
		return nil, errors.New("profession has no slug")
	}
	opts = opts.normalized()
	blocks, err := buildBlocks(p, opts)
	if err != nil {
		return nil, err
	}
	return &Routine{
		ID:             uuid.New(),
		ProfessionSlug: p.Slug,
		ProfessionName: p.Title,
		Title:          fmt.Sprintf("%s — daily routine", p.Title),
		Date:           opts.Date.Format("2006-01-02"),
		Intensity:      opts.Intensity,
		Blocks:         blocks,
		TotalMinutes:   totalMinutes(blocks),
		GeneratedAt:    time.Now(),
	}, nil
}
