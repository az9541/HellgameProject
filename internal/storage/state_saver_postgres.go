package storage

import (
	"HellgameProject/internal/engine"
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStateSaver struct {
	pool     *pgxpool.Pool
	saveName string
}

func NewPostgresPool(dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}

func NewPostgresStateSaver(pool *pgxpool.Pool, saveName string) *PostgresStateSaver {
	return &PostgresStateSaver{
		saveName: saveName,
		pool:     pool,
	}
}

func (s *PostgresStateSaver) Save(state *engine.WorldState) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	jsonGameState, err := json.Marshal(state)
	if err != nil {
		slog.Error("Failed to marshal game state to json", "err", err)
		return err
	}

	_, err = s.pool.Exec(ctx, sqlInsertGameSave, s.saveName, state.GlobalTick, jsonGameState)
	if err != nil {
		slog.Error("Failed to insert game state into Postgres", "err", err)
		return err
	}
	return nil
}

func (s *PostgresStateSaver) Load() (*engine.WorldState, error) {
	return nil, nil // TODO: Implement loading from Postgres
}
