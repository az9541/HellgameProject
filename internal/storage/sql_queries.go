package storage

const (
	sqlInsertGameSave = `INSERT INTO game_saves (save_name, tick, created_at, game_state)
	VALUES ($1, $2, NOW(), $3)`
)
