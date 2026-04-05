package migrations

import (
    "context"
    "embed"
    "fmt"
    "io/fs"
    "sort"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var sqlFiles embed.FS

func Run(ctx context.Context, pool *pgxpool.Pool) error {
    entries, err := fs.ReadDir(sqlFiles, ".")
    if err != nil {
        return fmt.Errorf("read migrations dir: %w", err)
    }

    var files []string
    for _, e := range entries {
        if strings.HasSuffix(e.Name(), ".sql") {
            files = append(files, e.Name())
        }
    }
    sort.Strings(files)

    for _, f := range files {
        data, err := sqlFiles.ReadFile(f)
        if err != nil {
            return fmt.Errorf("read %s: %w", f, err)
        }
        if _, err := pool.Exec(ctx, string(data)); err != nil {
            return fmt.Errorf("exec %s: %w", f, err)
        }
    }
    return nil
}