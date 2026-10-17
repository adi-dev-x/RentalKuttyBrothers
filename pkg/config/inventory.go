package config

import "github.com/google/uuid"

// InventoryID is the inventory every item and order is attached to. It is
// set once at startup by db.EnsureDefaultInventory and read everywhere else.
var InventoryID uuid.UUID
