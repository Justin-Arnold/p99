package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/justin/p99/internal/probe"
)

func WriteJSON(w io.Writer, result probe.RunResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func ReadJSON(path string) (probe.RunResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return probe.RunResult{}, err
	}
	defer f.Close()
	var result probe.RunResult
	if err := json.NewDecoder(f).Decode(&result); err != nil {
		return probe.RunResult{}, err
	}
	return result, nil
}

func WriteJSONFile(path string, result probe.RunResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteJSON(f, result)
}
