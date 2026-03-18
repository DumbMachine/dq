package annotations

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/pkg/types"
	"gopkg.in/yaml.v3"
)

func AnnotationsDir() string {
	return filepath.Join(config.ConfigDir(), "annotations")
}

func AnnotationPath(connection string) string {
	return filepath.Join(AnnotationsDir(), connection+".yaml")
}

func Load(connection string) (*types.AnnotationFile, error) {
	path := AnnotationPath(connection)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.AnnotationFile{Tables: make(map[string]types.AnnotationTable)}, nil
		}
		return nil, fmt.Errorf("reading annotations: %w", err)
	}

	var af types.AnnotationFile
	if err := yaml.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("parsing annotations: %w", err)
	}
	if af.Tables == nil {
		af.Tables = make(map[string]types.AnnotationTable)
	}
	return &af, nil
}

func Save(connection string, af *types.AnnotationFile) error {
	dir := AnnotationsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating annotations dir: %w", err)
	}

	data, err := yaml.Marshal(af)
	if err != nil {
		return fmt.Errorf("marshaling annotations: %w", err)
	}

	return os.WriteFile(AnnotationPath(connection), data, 0644)
}

func SetTableNote(connection, table, note string) error {
	af, err := Load(connection)
	if err != nil {
		return err
	}

	t := af.Tables[table]
	t.Note = note
	if t.Columns == nil {
		t.Columns = make(map[string]types.AnnotationColumn)
	}
	af.Tables[table] = t

	return Save(connection, af)
}

func SetColumnNote(connection, table, column, note string) error {
	af, err := Load(connection)
	if err != nil {
		return err
	}

	t := af.Tables[table]
	if t.Columns == nil {
		t.Columns = make(map[string]types.AnnotationColumn)
	}
	t.Columns[column] = types.AnnotationColumn{Note: note}
	af.Tables[table] = t

	return Save(connection, af)
}

func RemoveTableNote(connection, table string) error {
	af, err := Load(connection)
	if err != nil {
		return err
	}

	delete(af.Tables, table)
	return Save(connection, af)
}

func RemoveColumnNote(connection, table, column string) error {
	af, err := Load(connection)
	if err != nil {
		return err
	}

	t, ok := af.Tables[table]
	if !ok {
		return nil
	}
	delete(t.Columns, column)
	af.Tables[table] = t

	return Save(connection, af)
}

// GetTableAnnotation returns annotation for a specific table, or nil if not found.
func GetTableAnnotation(connection, table string) *types.TableAnnotation {
	af, err := Load(connection)
	if err != nil {
		return nil
	}

	t, ok := af.Tables[table]
	if !ok {
		return nil
	}

	ann := &types.TableAnnotation{
		Table:   t.Note,
		Columns: make(map[string]string),
	}
	for col, c := range t.Columns {
		ann.Columns[col] = c.Note
	}
	if ann.Table == "" && len(ann.Columns) == 0 {
		return nil
	}
	return ann
}
