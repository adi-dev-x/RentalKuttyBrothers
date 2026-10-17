package db

import (
	"database/sql"
	"fmt"
	"log"

	"myproject/pkg/config"
	"myproject/pkg/util/constants"
)

// EnsureDefaultInventory makes sure the default inventory row exists and
// stores its inventory_id in config.InventoryID. The upsert is keyed on the
// unique sitename, so restarts reuse the same row instead of adding new ones.
func EnsureDefaultInventory(conn *sql.DB) error {
	err := conn.QueryRow(`
		INSERT INTO inventory (sitename, status)
		VALUES ($1, 'ACTIVE')
		ON CONFLICT (sitename) DO UPDATE SET sitename = EXCLUDED.sitename
		RETURNING inventory_id`,
		constants.DefaultInventorySite,
	).Scan(&config.InventoryID)
	if err != nil {
		return fmt.Errorf("failed to ensure default inventory: %w", err)
	}

	log.Printf("using inventory %s (%s)", config.InventoryID, constants.DefaultInventorySite)
	return nil
}
