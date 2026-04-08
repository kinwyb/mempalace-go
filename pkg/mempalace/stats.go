package mempalace

import (
	"time"
)

// Stats represents palace statistics.
type Stats struct {
	TotalDocuments int
	TotalWings     int
	TotalRooms     int
	StorageSize    int64
	LastUpdated    time.Time
	Wings          map[string]WingStats
}

// WingStats represents statistics for a wing.
type WingStats struct {
	Name      string
	RoomCount int
	Total     int
	Rooms     map[string]int
}

// Wing represents a wing in the palace.
type Wing struct {
	Name        string
	Description string
	RoomCount   int
	DrawerCount int
}

// Room represents a room in a wing.
type Room struct {
	Name        string
	Wing        string
	Description string
	DrawerCount int
}