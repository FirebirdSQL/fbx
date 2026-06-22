package fbproto

import (
	"testing"
	"time"
)

func TestResolveLocationByTzRawOffsetUsesCanonicalName(t *testing.T) {
	tests := []struct {
		name          string
		tzRaw         uint16
		wantName      string
		wantOffsetSec int
	}{
		{
			name:          "UTC offset",
			tzRaw:         1439,
			wantName:      "+00:00",
			wantOffsetSec: 0,
		},
		{
			name:          "positive offset",
			tzRaw:         uint16(1439 + 4*60),
			wantName:      "+04:00",
			wantOffsetSec: 4 * 60 * 60,
		},
		{
			name:          "negative half-hour offset",
			tzRaw:         uint16(1439 - 5*60 - 30),
			wantName:      "-05:30",
			wantOffsetSec: -(5*60 + 30) * 60,
		},
		{
			name:          "positive quarter-hour offset",
			tzRaw:         uint16(1439 + 5*60 + 45),
			wantName:      "+05:45",
			wantOffsetSec: (5*60 + 45) * 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := ResolveLocationByTzRaw(tt.tzRaw)
			if loc.String() != tt.wantName {
				t.Fatalf("location name = %q, want %q", loc.String(), tt.wantName)
			}

			_, offsetSec := time.Date(2025, 1, 1, 0, 0, 0, 0, loc).Zone()
			if offsetSec != tt.wantOffsetSec {
				t.Fatalf("offset = %d, want %d", offsetSec, tt.wantOffsetSec)
			}
		})
	}
}

func TestResolveLocationByTzRawCachesLocations(t *testing.T) {
	tzRaw := uint16(1439 + 3*60)

	first := ResolveLocationByTzRaw(tzRaw)
	second := ResolveLocationByTzRaw(tzRaw)

	if first != second {
		t.Fatalf("ResolveLocationByTzRaw returned different location pointers for the same tzRaw")
	}
}
