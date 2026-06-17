package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Store) RunMigrations(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil && dir != "/migrations" {
		entries, err = os.ReadDir("/migrations")
	}
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil && dir != "/migrations" {
			path = filepath.Join("/migrations", name)
			content, err = os.ReadFile(path)
		}
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := s.db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
