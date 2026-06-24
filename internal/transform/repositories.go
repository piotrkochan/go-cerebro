package transform

import "encoding/json"

// Repository is one snapshot repository entry.
type Repository struct {
	Name     string          `json:"name" doc:"Repository name."`
	Type     string          `json:"type" doc:"Repository type (fs, s3, url, ...)."`
	Settings json.RawMessage `json:"settings" doc:"Repository settings, type-specific."`
}

// Repositories returns the repository list from a raw /_snapshot response. Port of models/repository/Repositories.scala.
func Repositories(raw json.RawMessage) ([]Repository, error) {
	var m map[string]struct {
		Type     string          `json:"type"`
		Settings json.RawMessage `json:"settings"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	out := make([]Repository, 0, len(m))
	for name, info := range m {
		settings := info.Settings
		if len(settings) == 0 {
			settings = json.RawMessage(`{}`)
		}
		out = append(out, Repository{Name: name, Type: info.Type, Settings: settings})
	}
	return out, nil
}
