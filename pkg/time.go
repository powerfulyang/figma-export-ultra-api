package pkg

import (
	"strconv"
	"strings"
	"time"
)

// timeUnit struct holds information for a single unit of time.
type timeUnit struct {
	// Name is the singular name of the unit (e.g., "day").
	Name string
	// ShortName is the compact representation (e.g., "d").
	ShortName string
	// Value is the duration of one unit in nanoseconds.
	Value time.Duration
}

// Pre-defined time units from largest to smallest for formatting logic.
// This slice is initialized only once when the package is loaded.
var units = []timeUnit{
	{Name: "day", ShortName: "d", Value: 24 * time.Hour},
	{Name: "hour", ShortName: "h", Value: time.Hour},
	{Name: "minute", ShortName: "m", Value: time.Minute},
	{Name: "second", ShortName: "s", Value: time.Second},
	{Name: "millisecond", ShortName: "ms", Value: time.Millisecond},
	{Name: "microsecond", ShortName: "μs", Value: time.Microsecond},
	{Name: "nanosecond", ShortName: "ns", Value: time.Nanosecond},
}

// SmartDurationFormat is a high-performance, dependency-free duration formatter.
func SmartDurationFormat(d time.Duration) string {
	// Handle the zero-value case explicitly for clarity.
	if d == 0 {
		return "0"
	}

	// Case 1: Duration is less than a second.
	// We find the largest appropriate unit (ms, µs, or ns) and display it.
	if d < time.Second {
		if d >= time.Millisecond {
			return strconv.FormatInt(d.Milliseconds(), 10) + "ms"
		}
		if d >= time.Microsecond {
			return strconv.FormatInt(d.Microseconds(), 10) + "μs"
		}
		// Fallback to nanoseconds.
		return strconv.FormatInt(d.Nanoseconds(), 10) + "ns"
	}

	// Case 2: Duration is one second or more.
	// We format up to 2 of the largest time units.
	var builder strings.Builder
	remaining := d
	parts := 0

	for _, unit := range units {
		// If the remaining duration is less than the current unit, skip to the next smaller unit.
		if remaining < unit.Value {
			continue
		}

		// Calculate how many of this unit fit into the remaining duration.
		count := remaining / unit.Value

		// Append the number and the short name to the builder.
		builder.WriteString(strconv.FormatInt(int64(count), 10))
		builder.WriteString(unit.ShortName)

		// Update the remaining duration.
		remaining %= unit.Value
		parts++

		// If we have our 2 parts, or if there's no remainder, we are done.
		if parts == 2 || remaining == 0 {
			break
		}
	}

	return builder.String()
}
