package cmd

import (
	"fmt"
	"time"

	"github.com/dumbmachine/db-cli/internal/annotations"
	"github.com/dumbmachine/db-cli/internal/cache"
	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/dumbmachine/db-cli/pkg/types"
	"github.com/spf13/cobra"

	_ "github.com/dumbmachine/db-cli/internal/database/mysql"
	_ "github.com/dumbmachine/db-cli/internal/database/postgres"
	_ "github.com/dumbmachine/db-cli/internal/database/sqlite"
)

var discoverRefresh bool

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Full database overview — schemas, tables, columns, FKs, row counts",
	Long:  `The "sidebar" command for agent orientation. Returns a hierarchical view of the entire database. Results are cached; use --refresh to re-introspect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		connName := GetConnection(cmd)

		// Try cache first (unless --refresh)
		if !discoverRefresh {
			cached, err := cache.LoadDiscover(connName)
			if err == nil && cached != nil {
				// Merge annotations into cached result
				mergeAnnotations(connName, cached)
				return output.Print(GetOutputFormat(), cached)
			}
		}

		// Introspect from live DB
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		conn, err := cfg.GetConnection(connName)
		if err != nil {
			return err
		}

		drv, err := database.Get(conn.Type)
		if err != nil {
			return err
		}

		db, err := drv.Connect(conn)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemas, err := drv.ListSchemas(db)
		if err != nil {
			return fmt.Errorf("listing schemas: %w", err)
		}

		result := &types.DiscoverResult{
			Connection: connName,
			Database:   conn.Database,
			CachedAt:   time.Now().UTC(),
		}

		for _, schemaName := range schemas {
			tables, err := drv.ListTables(db, schemaName)
			if err != nil {
				return fmt.Errorf("listing tables in %s: %w", schemaName, err)
			}

			schemaOverview := types.SchemaOverview{Name: schemaName}

			for _, tbl := range tables {
				columns, _ := drv.ListColumns(db, schemaName, tbl.Name)
				fks, _ := drv.ListForeignKeys(db, schemaName, tbl.Name)
				indexes, _ := drv.ListIndexes(db, schemaName, tbl.Name)

				tableOverview := types.TableOverview{
					Name: tbl.Name,
					Type: tbl.Type,
					// RowCount:    tbl.RowCount,
					Size:        tbl.Size,
					SizeBytes:   tbl.SizeBytes,
					Columns:     columns,
					ForeignKeys: fks,
					Indexes:     indexes,
				}

				schemaOverview.Tables = append(schemaOverview.Tables, tableOverview)
			}

			result.Schemas = append(result.Schemas, schemaOverview)
		}

		// Cache the result
		if err := cache.SaveDiscover(connName, result); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), `{"warning":"failed to cache discover result: %s"}`+"\n", err)
		}

		// Merge annotations
		mergeAnnotations(connName, result)

		return output.Print(GetOutputFormat(), result)
	},
}

func mergeAnnotations(connName string, result *types.DiscoverResult) {
	af, err := annotations.Load(connName)
	if err != nil || len(af.Tables) == 0 {
		return
	}

	for si := range result.Schemas {
		for ti := range result.Schemas[si].Tables {
			tableName := result.Schemas[si].Tables[ti].Name
			at, ok := af.Tables[tableName]
			if !ok {
				continue
			}

			ann := &types.TableAnnotation{
				Table:   at.Note,
				Columns: make(map[string]string),
			}
			for col, c := range at.Columns {
				ann.Columns[col] = c.Note
			}
			if ann.Table != "" || len(ann.Columns) > 0 {
				result.Schemas[si].Tables[ti].Annotations = ann
			}
		}
	}
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverRefresh, "refresh", false, "Force re-introspection from live database")
	rootCmd.AddCommand(discoverCmd)
}
