package fbproto

import (
	"fmt"
	"sync"
	"time"
	_ "time/tzdata"
)

var timezoneIDByName = func() map[string]uint16 {
	m := make(map[string]uint16, len(timezoneNameByID))
	for id, name := range timezoneNameByID {
		m[name] = uint16(id)
	}
	return m
}()

var timezoneLocationByRaw sync.Map

type timezoneLocationOverride struct {
	locationName  string
	offsetSeconds int
	unsupported   bool
}

var timezoneLocationOverrides = map[string]timezoneLocationOverride{
	"ACT":                      {locationName: "Australia/Darwin"},
	"AET":                      {locationName: "Australia/Sydney"},
	"AGT":                      {locationName: "America/Argentina/Buenos_Aires"},
	"ART":                      {locationName: "Africa/Cairo"},
	"AST":                      {locationName: "America/Anchorage"},
	"BET":                      {locationName: "America/Sao_Paulo"},
	"BST":                      {locationName: "Asia/Dhaka"},
	"CAT":                      {locationName: "Africa/Maputo"},
	"CNT":                      {locationName: "America/St_Johns"},
	"CST":                      {locationName: "America/Chicago"},
	"CTT":                      {locationName: "Asia/Shanghai"},
	"Canada/East-Saskatchewan": {locationName: "America/Regina"},
	"EAT":                      {locationName: "Africa/Addis_Ababa"},
	"ECT":                      {locationName: "Europe/Paris"},
	"IET":                      {locationName: "America/Indiana/Indianapolis"},
	"IST":                      {locationName: "Asia/Kolkata"},
	"JST":                      {locationName: "Asia/Tokyo"},
	"MIT":                      {locationName: "Pacific/Apia"},
	"NET":                      {locationName: "Asia/Yerevan"},
	"NST":                      {locationName: "Pacific/Auckland"},
	"PLT":                      {locationName: "Asia/Karachi"},
	"PNT":                      {locationName: "America/Phoenix"},
	"PRT":                      {locationName: "America/Puerto_Rico"},
	"PST":                      {locationName: "America/Los_Angeles"},
	"SST":                      {locationName: "Pacific/Guadalcanal"},
	"SystemV/AST4":             {offsetSeconds: -4 * 60 * 60},
	"SystemV/AST4ADT":          {unsupported: true},
	"SystemV/CST6":             {offsetSeconds: -6 * 60 * 60},
	"SystemV/CST6CDT":          {unsupported: true},
	"SystemV/EST5":             {offsetSeconds: -5 * 60 * 60},
	"SystemV/EST5EDT":          {unsupported: true},
	"SystemV/HST10":            {offsetSeconds: -10 * 60 * 60},
	"SystemV/MST7":             {offsetSeconds: -7 * 60 * 60},
	"SystemV/MST7MDT":          {unsupported: true},
	"SystemV/PST8":             {offsetSeconds: -8 * 60 * 60},
	"SystemV/PST8PDT":          {unsupported: true},
	"SystemV/YST9":             {offsetSeconds: -9 * 60 * 60},
	"SystemV/YST9YDT":          {unsupported: true},
	"US/Pacific-New":           {locationName: "America/Los_Angeles"},
	"VST":                      {locationName: "Asia/Ho_Chi_Minh"},
}

func ResolveTzRawByLocationName(name string) (uint16, bool) {
	tzRaw, ok := timezoneIDByName[name]
	return tzRaw, ok
}

// ResolveLocationNameByTzRaw returns the Firebird name registered for tzRaw.
func ResolveLocationNameByTzRaw(tzRaw uint16) (string, bool) {
	name, ok := timezoneNameByID[int(tzRaw)]
	return name, ok
}

func ResolveLocationByTzRaw(tzRaw uint16) (*time.Location, error) {
	if loc, ok := timezoneLocationByRaw.Load(tzRaw); ok {
		return loc.(*time.Location), nil
	}

	loc, err := resolveLocationByTzRaw(tzRaw)
	if err != nil {
		return nil, err
	}

	actual, _ := timezoneLocationByRaw.LoadOrStore(tzRaw, loc)
	return actual.(*time.Location), nil
}

func resolveLocationByTzRaw(tzRaw uint16) (*time.Location, error) {
	if tzRaw <= 2878 {
		offsetMinutes := int(tzRaw) - 1439
		return time.FixedZone(formatTimezoneOffsetName(offsetMinutes), offsetMinutes*60), nil
	}

	name, ok := ResolveLocationNameByTzRaw(tzRaw)
	if !ok {
		return nil, fmt.Errorf("unknown Firebird time zone ID %d", tzRaw)
	}

	if override, ok := timezoneLocationOverrides[name]; ok {
		if override.unsupported {
			return nil, fmt.Errorf("time zone %q is not supported by fbx", name)
		}
		if override.locationName == "" {
			return time.FixedZone(name, override.offsetSeconds), nil
		}
		name = override.locationName
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("cannot load Firebird time zone %q: %w", name, err)
	}

	return loc, nil
}

func formatTimezoneOffsetName(offsetMinutes int) string {
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
		offsetMinutes = -offsetMinutes
	}

	return fmt.Sprintf("%s%02d:%02d", sign, offsetMinutes/60, offsetMinutes%60)
}
