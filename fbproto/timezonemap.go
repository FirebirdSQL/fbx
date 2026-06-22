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

func getTimezoneNameByID(id int) string {
	return timezoneNameByID[id]
}

func ResolveTzRawByLocationName(name string) (uint16, bool) {
	tzRaw, ok := timezoneIDByName[name]
	return tzRaw, ok
}

func ResolveLocationByTzRaw(tzRaw uint16) *time.Location {
	if loc, ok := timezoneLocationByRaw.Load(tzRaw); ok {
		return loc.(*time.Location)
	}

	loc := resolveLocationByTzRaw(tzRaw)
	actual, _ := timezoneLocationByRaw.LoadOrStore(tzRaw, loc)
	return actual.(*time.Location)
}

func resolveLocationByTzRaw(tzRaw uint16) *time.Location {
	if tzRaw <= 2878 {
		offsetMinutes := int(tzRaw) - 1439
		return time.FixedZone(formatTimezoneOffsetName(offsetMinutes), offsetMinutes*60)
	}

	name := getTimezoneNameByID(int(tzRaw))
	if name == "" {
		return time.UTC
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return loc
}

func formatTimezoneOffsetName(offsetMinutes int) string {
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
		offsetMinutes = -offsetMinutes
	}

	return fmt.Sprintf("%s%02d:%02d", sign, offsetMinutes/60, offsetMinutes%60)
}
