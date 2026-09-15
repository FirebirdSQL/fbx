package fbproto

import (
	"sort"
	"testing"
	"time"
)

func TestResolveLocationByTzRawResolvesAllRegisteredZones(t *testing.T) {
	unsupported := map[string]bool{
		"SystemV/AST4ADT": true,
		"SystemV/CST6CDT": true,
		"SystemV/EST5EDT": true,
		"SystemV/MST7MDT": true,
		"SystemV/PST8PDT": true,
		"SystemV/YST9YDT": true,
	}

	ids := make([]int, 0, len(timezoneNameByID))
	for id := range timezoneNameByID {
		ids = append(ids, id)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))

	for _, id := range ids {
		name := timezoneNameByID[id]

		tzRaw, ok := ResolveTzRawByLocationName(name)
		if !ok {
			t.Errorf("time zone %d %q is missing from the name-to-ID mapping", id, name)
			continue
		}
		if tzRaw != uint16(id) {
			t.Errorf("time zone %q resolves to ID %d, want %d", name, tzRaw, id)
			continue
		}

		loc := ResolveLocationByTzRaw(uint16(id))
		if unsupported[name] {
			if loc != time.UTC {
				t.Errorf("unsupported time zone %d %q resolved as %q, want UTC fallback", id, name, loc)
			}
			delete(unsupported, name)
			continue
		}
		if loc == time.UTC && name != "UTC" {
			t.Errorf("time zone %d %q resolved to the UTC fallback", id, name)
		}
	}

	for name := range unsupported {
		t.Errorf("expected unsupported time zone %q is not registered", name)
	}
}

func TestResolveLocationByTzRawAliases(t *testing.T) {
	aliases := map[string]string{
		"ACT":                      "Australia/Darwin",
		"AET":                      "Australia/Sydney",
		"AGT":                      "America/Argentina/Buenos_Aires",
		"ART":                      "Africa/Cairo",
		"AST":                      "America/Anchorage",
		"BET":                      "America/Sao_Paulo",
		"BST":                      "Asia/Dhaka",
		"CAT":                      "Africa/Maputo",
		"CNT":                      "America/St_Johns",
		"CST":                      "America/Chicago",
		"CTT":                      "Asia/Shanghai",
		"Canada/East-Saskatchewan": "America/Regina",
		"EAT":                      "Africa/Addis_Ababa",
		"ECT":                      "Europe/Paris",
		"IET":                      "America/Indiana/Indianapolis",
		"IST":                      "Asia/Kolkata",
		"JST":                      "Asia/Tokyo",
		"MIT":                      "Pacific/Apia",
		"NET":                      "Asia/Yerevan",
		"NST":                      "Pacific/Auckland",
		"PLT":                      "Asia/Karachi",
		"PNT":                      "America/Phoenix",
		"PRT":                      "America/Puerto_Rico",
		"PST":                      "America/Los_Angeles",
		"SST":                      "Pacific/Guadalcanal",
		"US/Pacific-New":           "America/Los_Angeles",
		"VST":                      "Asia/Ho_Chi_Minh",
	}

	for name, wantLocation := range aliases {
		t.Run(name, func(t *testing.T) {
			tzRaw, ok := ResolveTzRawByLocationName(name)
			if !ok {
				t.Fatalf("time zone %q is not registered", name)
			}

			loc := ResolveLocationByTzRaw(tzRaw)
			if loc.String() != wantLocation {
				t.Fatalf("location = %q, want %q", loc, wantLocation)
			}
		})
	}
}

func TestResolveLocationByTzRawFixedSystemVZones(t *testing.T) {
	zones := map[string]int{
		"SystemV/AST4":  -4 * 60 * 60,
		"SystemV/CST6":  -6 * 60 * 60,
		"SystemV/EST5":  -5 * 60 * 60,
		"SystemV/HST10": -10 * 60 * 60,
		"SystemV/MST7":  -7 * 60 * 60,
		"SystemV/PST8":  -8 * 60 * 60,
		"SystemV/YST9":  -9 * 60 * 60,
	}

	for name, wantOffset := range zones {
		t.Run(name, func(t *testing.T) {
			tzRaw, ok := ResolveTzRawByLocationName(name)
			if !ok {
				t.Fatalf("time zone %q is not registered", name)
			}

			loc := ResolveLocationByTzRaw(tzRaw)
			if loc.String() != name {
				t.Fatalf("location = %q, want %q", loc, name)
			}
			_, offset := time.Date(2025, 1, 1, 0, 0, 0, 0, loc).Zone()
			if offset != wantOffset {
				t.Fatalf("offset = %d, want %d", offset, wantOffset)
			}
		})
	}
}

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
