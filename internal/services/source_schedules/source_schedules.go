package source_schedules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/walens/walens/internal/dbtypes"
)

// ErrDBUnavailable is returned when the database is not available.
var ErrDBUnavailable = errors.New("database unavailable")

// ErrScheduleNotFound is returned when the requested source schedule does not exist.
var ErrScheduleNotFound = errors.New("source schedule not found")

// ErrSourceNotFound is returned when the referenced source does not exist.
var ErrSourceNotFound = errors.New("source not found")

// ErrInvalidCronExpr is returned when the cron expression is invalid.
var ErrInvalidCronExpr = errors.New("invalid cron expression")

// ErrSchedulerUnavailable is returned when the scheduler is not available.
var ErrSchedulerUnavailable = errors.New("scheduler unavailable")

// ScheduleRow represents a source schedule row in the database.
type ScheduleRow struct {
	ID        dbtypes.UUID          `json:"id" doc:"Unique source schedule identifier (UUIDv7)."`
	SourceID  dbtypes.UUID          `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string                `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled dbtypes.BoolInt       `json:"is_enabled" doc:"Whether this schedule is active."`
	CreatedAt dbtypes.UnixMilliTime `json:"created_at" doc:"Schedule creation timestamp."`
	UpdatedAt dbtypes.UnixMilliTime `json:"updated_at" doc:"Last modification timestamp."`
}

// ScheduleWithSource includes the source name for overlap checking.
type ScheduleWithSource struct {
	ScheduleRow
	SourceName string `json:"source_name" doc:"Name of the parent source for overlap grouping."`
}

// OverlapWarning represents a proximity warning between two schedules.
type OverlapWarning struct {
	ScheduleID1  dbtypes.UUID `json:"schedule_id_1" doc:"First schedule ID."`
	ScheduleID2  dbtypes.UUID `json:"schedule_id_2" doc:"Second schedule ID."`
	SourceName   string       `json:"source_name" doc:"Source name that both schedules belong to."`
	Occurrence1  time.Time    `json:"occurrence_1" doc:"First occurrence that triggered the warning."`
	Occurrence2  time.Time    `json:"occurrence_2" doc:"Second occurrence that triggered the warning."`
	DistanceMins float64      `json:"distance_mins" doc:"Distance between occurrences in minutes."`
}

// CreateScheduleInput contains the fields needed to create a new source schedule.
type CreateScheduleInput struct {
	SourceID  string `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this schedule is active."`
}

// UpdateScheduleInput contains the fields needed to update an existing source schedule.
// All fields are required for full-object update semantics.
type UpdateScheduleInput struct {
	ID        string `json:"id" doc:"Unique source schedule identifier."`
	SourceID  string `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this schedule is active."`
}

// Service provides CRUD operations for source schedules.
type Service struct {
	db        *sql.DB
	scheduler SchedulerInterface
}

// SchedulerInterface defines the scheduler methods used by the service.
type SchedulerInterface interface {
	Reload() error
}

// NewService creates a new source_schedules service.
func NewService(db *sql.DB, scheduler SchedulerInterface) *Service {
	return &Service{db: db, scheduler: scheduler}
}

// ValidateCronExpr validates a 5-field cron expression and returns the parsed schedule.
// Uses standard cron format: minute hour day month weekday
func ValidateCronExpr(expr string) (cron.Schedule, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser.Parse(expr)
}

// CheckOverlapWarnings checks for schedule overlaps within a bounded lookahead window.
// Returns warnings for any schedules of the same source that have occurrences less than 5 minutes apart.
func CheckOverlapWarnings(schedules []ScheduleWithSource, lookaheadDays int) ([]OverlapWarning, error) {
	if len(schedules) < 2 {
		return nil, nil
	}

	warnings := make([]OverlapWarning, 0)
	now := time.Now().UTC()
	end := now.AddDate(0, 0, lookaheadDays)
	minDistance := 5 * time.Minute

	// Group schedules by source name
	bySource := make(map[string][]ScheduleWithSource)
	for _, sched := range schedules {
		bySource[sched.SourceName] = append(bySource[sched.SourceName], sched)
	}

	for sourceName, sourceSchedules := range bySource {
		if len(sourceSchedules) < 2 {
			continue
		}

		// Parse all cron expressions
		type parsedSchedule struct {
			schedule ScheduleWithSource
			parser   cron.Schedule
		}
		parsed := make([]parsedSchedule, 0, len(sourceSchedules))
		for _, s := range sourceSchedules {
			parser, err := ValidateCronExpr(s.CronExpr)
			if err != nil {
				continue // Skip invalid expressions
			}
			parsed = append(parsed, parsedSchedule{schedule: s, parser: parser})
		}

		if len(parsed) < 2 {
			continue
		}

		// Generate occurrences for each schedule within the window
		type occurrence struct {
			schedID  dbtypes.UUID
			t        time.Time
			schedIdx int
		}
		allOccurrences := make([]occurrence, 0)

		for idx, p := range parsed {
			for t := p.parser.Next(now); t.Before(end); t = p.parser.Next(t) {
				allOccurrences = append(allOccurrences, occurrence{
					schedID:  p.schedule.ID,
					t:        t,
					schedIdx: idx,
				})
			}
		}

		// Sort by time
		sort.Slice(allOccurrences, func(i, j int) bool {
			return allOccurrences[i].t.Before(allOccurrences[j].t)
		})

		// Check adjacent occurrences from different schedules
		for i := 1; i < len(allOccurrences); i++ {
			prev := allOccurrences[i-1]
			curr := allOccurrences[i]

			if prev.schedIdx == curr.schedIdx {
				continue // Same schedule, skip
			}

			dist := curr.t.Sub(prev.t)
			if dist < minDistance {
				// Find the schedules for warning message
				var sched1, sched2 ScheduleWithSource
				for _, p := range parsed {
					if p.schedule.ID == prev.schedID {
						sched1 = p.schedule
					}
					if p.schedule.ID == curr.schedID {
						sched2 = p.schedule
					}
				}
				warnings = append(warnings, OverlapWarning{
					ScheduleID1:  sched1.ID,
					ScheduleID2:  sched2.ID,
					SourceName:   sourceName,
					Occurrence1:  prev.t,
					Occurrence2:  curr.t,
					DistanceMins: dist.Minutes(),
				})
			}
		}
	}

	return warnings, nil
}

// ListSchedules returns all source schedules.
func (s *Service) ListSchedules(ctx context.Context) ([]ScheduleRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, source_id, cron_expr, is_enabled, created_at, updated_at
		FROM source_schedules
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query source_schedules: %w", err)
	}
	defer rows.Close()

	results := make([]ScheduleRow, 0)
	for rows.Next() {
		var sched ScheduleRow
		if err := rows.Scan(
			&sched.ID, &sched.SourceID, &sched.CronExpr,
			&sched.IsEnabled, &sched.CreatedAt, &sched.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule row: %w", err)
		}
		results = append(results, sched)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// ListSchedulesWithSourceName returns all source schedules with their source names.
func (s *Service) ListSchedulesWithSourceName(ctx context.Context) ([]ScheduleWithSource, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT ss.id, ss.source_id, ss.cron_expr, ss.is_enabled, ss.created_at, ss.updated_at, s.name
		FROM source_schedules ss
		JOIN sources s ON s.id = ss.source_id
		ORDER BY ss.created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query source_schedules: %w", err)
	}
	defer rows.Close()

	results := make([]ScheduleWithSource, 0)
	for rows.Next() {
		var sched ScheduleWithSource
		if err := rows.Scan(
			&sched.ID, &sched.SourceID, &sched.CronExpr,
			&sched.IsEnabled, &sched.CreatedAt, &sched.UpdatedAt,
			&sched.SourceName,
		); err != nil {
			return nil, fmt.Errorf("scan schedule row: %w", err)
		}
		results = append(results, sched)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetSchedule returns a single source schedule by ID.
func (s *Service) GetSchedule(ctx context.Context, id dbtypes.UUID) (*ScheduleRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, source_id, cron_expr, is_enabled, created_at, updated_at
		FROM source_schedules
		WHERE id = ?
	`

	var sched ScheduleRow
	err := s.db.QueryRowContext(ctx, query, id.UUID.String()).Scan(
		&sched.ID, &sched.SourceID, &sched.CronExpr,
		&sched.IsEnabled, &sched.CreatedAt, &sched.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrScheduleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query schedule: %w", err)
	}

	return &sched, nil
}

// sourceExists checks if a source with the given ID exists.
func (s *Service) sourceExists(ctx context.Context, sourceID dbtypes.UUID) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sources WHERE id = ?`, sourceID.UUID.String()).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check source exists: %w", err)
	}
	return true, nil
}

// CreateSchedule creates a new source schedule.
func (s *Service) CreateSchedule(ctx context.Context, input *CreateScheduleInput) (*ScheduleRow, []OverlapWarning, error) {
	if s.db == nil {
		return nil, nil, ErrDBUnavailable
	}

	// Validate cron expression
	if _, err := ValidateCronExpr(input.CronExpr); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}

	// Parse and validate source ID
	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source ID format: %w", err)
	}

	// Check source exists
	exists, err := s.sourceExists(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrSourceNotFound
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		id.UUID.String(), sourceID.UUID.String(), input.CronExpr,
		isEnabled, now, now,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert schedule: %w", err)
	}

	// Trigger scheduler reload
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}

	// Check for overlap warnings after creation
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return &ScheduleRow{
			ID:        id,
			SourceID:  sourceID,
			CronExpr:  input.CronExpr,
			IsEnabled: isEnabled,
			CreatedAt: now,
			UpdatedAt: now,
		}, warnings, nil
	}

	return &ScheduleRow{
		ID:        id,
		SourceID:  sourceID,
		CronExpr:  input.CronExpr,
		IsEnabled: isEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil, nil
}

// UpdateSchedule updates an existing source schedule with full-object update semantics.
func (s *Service) UpdateSchedule(ctx context.Context, input *UpdateScheduleInput) (*ScheduleRow, []OverlapWarning, error) {
	if s.db == nil {
		return nil, nil, ErrDBUnavailable
	}

	// Validate cron expression
	if _, err := ValidateCronExpr(input.CronExpr); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}

	// Parse IDs
	id, err := dbtypes.NewUUIDFromString(input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid schedule ID format: %w", err)
	}

	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source ID format: %w", err)
	}

	// Check schedule exists
	existing, err := s.GetSchedule(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	// Check source exists
	exists, err := s.sourceExists(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrSourceNotFound
	}

	now := dbtypes.NewUnixMilliTimeNow()
	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		UPDATE source_schedules
		SET source_id = ?, cron_expr = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		sourceID.UUID.String(), input.CronExpr, isEnabled, now, id.UUID.String(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update schedule: %w", err)
	}

	// Trigger scheduler reload
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}

	// Check for overlap warnings after update
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return &ScheduleRow{
			ID:        id,
			SourceID:  sourceID,
			CronExpr:  input.CronExpr,
			IsEnabled: isEnabled,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: now,
		}, warnings, nil
	}

	return &ScheduleRow{
		ID:        id,
		SourceID:  sourceID,
		CronExpr:  input.CronExpr,
		IsEnabled: isEnabled,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}, nil, nil
}

// DeleteSchedule deletes a source schedule by ID.
func (s *Service) DeleteSchedule(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}

	// Check schedule exists
	_, err := s.GetSchedule(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM source_schedules WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, id.UUID.String())
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}

	// Trigger scheduler reload
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}

	return nil
}
