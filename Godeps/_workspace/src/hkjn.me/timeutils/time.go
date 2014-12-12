// Package timeutils provides some convenience functions around time.
package timeutils // import "hkjn.me/timeutils"

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	StdLayout     = "2006-01-02 15:04" // a simple time layout
	EmptyTimeZone = TimeZone("")       // the empty timezone
)

// TimeZone is a string that can be given to time.LoadLocation.
type TimeZone string

// Day describes a specific day and can be used as map keys.
type Day struct {
	Year, Day int
	Month     time.Month
	TimeZone  TimeZone
}

// DayFromTime returns the equivalent day object from time.
func DayFromTime(t time.Time) Day {
	year, month, day := t.Date()
	return Day{year, day, month, TimeZone(t.Location().String())}
}

// ToTime returns the time object representing the start of the day.
func (d Day) ToTime() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, MustLoadLoc(d.TimeZone))
}

// String returns a description of the day object.
func (d Day) String() string {
	return d.ToTime().Format("2006-01-02 MST")
}

// String returns a description of the day object without the TZ.
func (d Day) ShortString() string {
	return d.ToTime().Format("2006-01-02")
}

// MustLoadLoc loads the time.Location specified by the string, or panics.
func MustLoadLoc(tz TimeZone) *time.Location {
	loc, err := time.LoadLocation(string(tz))
	if err != nil {
		log.Fatalf("bad location %q: %v\n", tz, err)
	}
	return loc
}

// MustParseDuration returns the time.Duration specified by the string, or panics.
func MustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("bad duration %q: %v\n", s, err)
	}
	return d
}

// Must panics if error is non-nil.
//
// Intended use is to wrap function calls that must succeed, e.g:
//   t := Must(ParseStd("2013-07-31"))
func Must(t time.Time, err error) time.Time {
	if err != nil {
		log.Fatalf("got err: %v\n", err)
	}
	return t
}

// AsMillis returns the number of milliseconds (in offset by sec) for the time.
func AsMillis(t time.Time, offset int) int {
	return int(t.UTC().UnixNano()/1000000) + offset*1000
}

// ParseStd parses the value using a standard layout, with time.UTC timezone.
func ParseStd(value string) (time.Time, error) {
	return time.Parse(StdLayout, value)
}

// daysIn returns the number of days in a month for a given year.
func daysIn(m time.Month, year int) int {
	// This is equivalent to the unexported time.daysIn(m, year).
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Weekday gets the Monday-indexed number for the time.Weekday.
func Weekday(d time.Weekday) int {
	day := (d - 1) % 7
	if day < 0 {
		day += 7
	}
	return int(day)
}

// StartOfWeek returns the start of the current week for the time.
func StartOfWeek(t time.Time) time.Time {
	// Figure out number of days to back up until Mon:
	// Sun is 0 -> 6, Sat is 6 -> 5, etc.
	toMon := Weekday(t.Weekday())
	y, m, d := t.AddDate(0, 0, -int(toMon)).Date()
	// Result is 00:00:00 on that year, month, day.
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// Parse extracts time from string-based info, with some constraints.
//
// The described time cannot be in the future, or more than 1000 years in the past.
//
// Note that month is 0-indexed, unlike time.Month.
func Parse(year, month, day, hourMinute string, loc *time.Location) (time.Time, error) {
	now := time.Now().In(loc)

	y64, err := strconv.ParseInt(year, 10, 0)
	y := int(y64)
	if err != nil {
		return time.Time{}, err
	}
	if y < now.Year()-1000 {
		return time.Time{}, fmt.Errorf("bad year; %d is too far in the past", y)
	}
	m, err := strconv.ParseInt(month, 10, 0)
	if err != nil {
		return time.Time{}, err
	}
	if m < 0 || m > 11 {
		return time.Time{}, fmt.Errorf("bad month: %d is not within [0, 11]", m)
	}
	// Month +1 since time.Month is [1, 12].
	m = m + 1
	d64, err := strconv.ParseInt(day, 10, 0)
	d := int(d64)
	if err != nil {
		return time.Time{}, err
	}
	if d < 1 {
		return time.Time{}, fmt.Errorf("bad day: %d; can't be negative", d)
	} else if d > daysIn(time.Month(m), y) {
		return time.Time{}, fmt.Errorf("bad day: %d; only %d days in %v, %d", d, daysIn(time.Month(m), y), time.Month(m), y)
	}
	parts := strings.Split(hourMinute, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("bad hour/minute: %s", hourMinute)
	}
	h, err := strconv.ParseInt(parts[0], 10, 0)
	if err != nil {
		return time.Time{}, err
	}
	if h < 0 || h > 60 {
		return time.Time{}, fmt.Errorf("bad hour: %d", h)
	}
	min, err := strconv.ParseInt(parts[1], 10, 0)
	if err != nil {
		return time.Time{}, err
	}
	if min < 0 || min > 60 {
		return time.Time{}, fmt.Errorf("bad minute: %d", min)
	}

	t := time.Time(time.Date(int(y), time.Month(m), int(d), int(h), int(min), 0, 0, loc))
	if t.After(now) {
		return time.Time{}, fmt.Errorf("bad time; %v is in the future", time.Time(t))
	}
	return t, nil
}

// Selector holds info useful to make time selections.
type Selector struct {
	SelectedDay   int
	SelectedMonth time.Month
	SelectedYear  int
	SelectedTime  string
	Months        []time.Month
	Years         []int
	DaysInMonth   []int
}

// Create populates a Selector from given starting point.
func (s *Selector) Create(from time.Time) {
	days := make([]int, 31)
	for d := 0; d < 31; d++ { // TODO: Actual number of days / month (change dynamically on selection?).
		days[d] = d + 1
	}
	numYears := 5
	years := make([]int, numYears)
	for i := 0; i < numYears; i++ {
		years[i] = from.Year() - i
	}
	*s = Selector{
		SelectedYear:  from.Year(),
		SelectedMonth: from.Month() - 1, // -1 to give [0, 11]
		SelectedDay:   from.Day(),
		SelectedTime:  from.Format("15:04"),
		DaysInMonth:   days,
		Months:        make([]time.Month, 12),
		Years:         years,
	}
	for i := 1; i <= 12; i++ {
		s.Months[i-1] = time.Month(i)
	}
}

// DescDuration returns a human-readable string describing the duration.
func DescDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%0.1f sec ago", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%0.1f min ago", d.Minutes())
	} else if d < time.Hour*24 {
		return fmt.Sprintf("%0.1f hrs ago", d.Hours())
	} else {
		return fmt.Sprintf("%0.1f days ago", d.Hours()/24.0)
	}
}
