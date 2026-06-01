package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"StreamSignal/internal/domain"
)

type DestinationRepository struct {
	db *sql.DB
}

func NewDestinationRepository(db *sql.DB) *DestinationRepository {
	return &DestinationRepository{db: db}
}

func (r *DestinationRepository) List(ctx context.Context) ([]domain.Destination, error) {
	rows, err := r.db.QueryContext(ctx, `
select id, platform, name, enabled, template, config_json, created_at, updated_at
from destinations
order by name asc`)
	if err != nil {
		return nil, fmt.Errorf("list destinations: %w", err)
	}
	defer rows.Close()

	destinations := make([]domain.Destination, 0)
	for rows.Next() {
		destination, err := scanDestination(rows)
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, destination)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate destinations: %w", err)
	}

	return destinations, nil
}

func (r *DestinationRepository) Get(ctx context.Context, id string) (domain.Destination, error) {
	row := r.db.QueryRowContext(ctx, `
select id, platform, name, enabled, template, config_json, created_at, updated_at
from destinations
where id = ?`, id)

	destination, err := scanDestination(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Destination{}, err
		}
		return domain.Destination{}, fmt.Errorf("get destination %q: %w", id, err)
	}

	return destination, nil
}

func (r *DestinationRepository) Save(ctx context.Context, destination domain.Destination) error {
	_, err := r.db.ExecContext(ctx, `
insert into destinations(id, platform, name, enabled, template, config_json, created_at, updated_at)
values(?, ?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update set
  platform = excluded.platform,
  name = excluded.name,
  enabled = excluded.enabled,
  template = excluded.template,
  config_json = excluded.config_json,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at`,
		destination.ID,
		string(destination.Platform),
		destination.Name,
		boolToInt(destination.Enabled),
		destination.Template,
		destination.ConfigJSON,
		formatTime(destination.CreatedAt),
		formatTime(destination.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save destination %q: %w", destination.ID, err)
	}

	return nil
}

func (r *DestinationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `delete from destinations where id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete destination %q: %w", id, err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDestination(scanner rowScanner) (domain.Destination, error) {
	var destination domain.Destination
	var platform string
	var enabled int
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&destination.ID,
		&platform,
		&destination.Name,
		&enabled,
		&destination.Template,
		&destination.ConfigJSON,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return domain.Destination{}, err
	}

	parsedCreatedAt, err := parseTime(createdAt)
	if err != nil {
		return domain.Destination{}, fmt.Errorf("parse destination created_at: %w", err)
	}

	parsedUpdatedAt, err := parseTime(updatedAt)
	if err != nil {
		return domain.Destination{}, fmt.Errorf("parse destination updated_at: %w", err)
	}

	destination.Platform = domain.DestinationPlatform(platform)
	destination.Enabled = intToBool(enabled)
	destination.CreatedAt = parsedCreatedAt
	destination.UpdatedAt = parsedUpdatedAt

	return destination, nil
}
