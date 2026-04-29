package library

import (
	"encoding/json"
	"fmt"
	"os"
)

func Load(path string) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lib := &Library{}
	if err := json.Unmarshal(data, lib); err != nil {
		return nil, fmt.Errorf("invalid library.json: %w", err)
	}

	// Ensure all maps are initialized
	for i := range lib.Arcs {
		if lib.Arcs[i].Episodes == nil {
			lib.Arcs[i].Episodes = map[string]Episode{}
		}
	}

	return lib, nil
}
