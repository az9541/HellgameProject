package storage

import (
	"HellgameProject/internal/engine"
	"encoding/json"
	"log/slog"
	"os"
)

const defaultSavePath = "savegame.json"

type JSONStateSaver struct {
	filePath string
}

// Создаёт новый адаптер для сохранения состояния в JSON файл
func NewJSONStateSaver() *JSONStateSaver {
	return &JSONStateSaver{filePath: defaultSavePath}
}

func (s *JSONStateSaver) Save(state *engine.WorldState) error {
	file, err := os.Create(s.filePath)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("Failed to close save-file ", "err", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Для красивого форматирования JSON
	return encoder.Encode(state)
}

func (s *JSONStateSaver) Load() (*engine.WorldState, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("Failed to close save-file ", "err", err)
		}
	}()

	// Читаем JSON и декодируем его в структуру WorldState
	var state engine.WorldState
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}
