// Package timeinterval is time range data structure ,many trade system has this.
// code inspired by Altermanager
package timeinterval

import (
	"errors"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	title      = cases.Title(language.English)
	daysOfWeek = map[string]time.Weekday{
		time.Sunday.String():    time.Sunday,
		time.Monday.String():    time.Monday,
		time.Tuesday.String():   time.Tuesday,
		time.Wednesday.String(): time.Wednesday,
		time.Thursday.String():  time.Thursday,
		time.Friday.String():    time.Friday,
		time.Saturday.String():  time.Saturday,
	}
	months = map[string]time.Month{
		time.January.String():   time.January,
		time.February.String():  time.February,
		time.March.String():     time.March,
		time.April.String():     time.April,
		time.May.String():       time.May,
		time.June.String():      time.June,
		time.July.String():      time.July,
		time.August.String():    time.August,
		time.September.String(): time.September,
		time.October.String():   time.October,
		time.November.String():  time.November,
		time.December.String():  time.December,
	}
)

type (
	// A range with a Beginning and End that can be represented as strings.
	stringableRange interface {
		setBegin(int)
		setEnd(int)
		// Try to map a member of the range into an integer.
		memberFromString(string) (int, error)
	}

	TimeInterval struct {
		Times       []TimeRange       `yaml:"times,omitempty" json:"times,omitempty"`
		Weekdays    []WeekdayRange    `yaml:"weekdays,flow,omitempty" json:"weekdays,omitempty"`
		DaysOfMonth []DayOfMonthRange `yaml:"daysOfMonth,flow,omitempty" json:"daysOfMonth,omitempty"`
		Months      []MonthRange      `yaml:"months,flow,omitempty" json:"months,omitempty"`
		Years       []YearRange       `yaml:"years,flow,omitempty" json:"years,omitempty"`
		Location    *Location         `yaml:"location,flow,omitempty" json:"location,omitempty"`
	}

	// TimeRange represents a range of minutes within a 1440-minute day, exclusive of the End minute. A day consists of 1440 minutes.
	// For example, 4:00PM to End of the day would Begin at 1020 and End at 1440.
	TimeRange struct {
		StartMinute int
		EndMinute   int
	}
	// InclusiveRange is used to hold the Beginning and End values of many time interval components.
	InclusiveRange struct {
		Begin int
		End   int
	}
	// A WeekdayRange is an inclusive range between [0, 6] where 0 = Sunday.
	WeekdayRange struct {
		InclusiveRange
	}
	// A DayOfMonthRange is an inclusive range that may have negative Beginning/End values that represent distance from the End of the month Beginning at -1.
	DayOfMonthRange struct {
		InclusiveRange
	}
	// A MonthRange is an inclusive range between [1, 12] where 1 = January.
	MonthRange struct {
		InclusiveRange
	}
	// A YearRange is a positive inclusive range.
	YearRange struct {
		InclusiveRange
	}
	// A Location is a container for a time.Location, used for custom unmarshalling/validation logic.
	Location struct {
		*time.Location
	}
)

func (tz *Location) UnmarshalText(str []byte) error {
	loc, err := time.LoadLocation(string(str))
	if err != nil {
		if runtime.GOOS == "windows" {
			if zoneinfo := os.Getenv("ZONEINFO"); zoneinfo != "" {
				return fmt.Errorf("%w (ZONEINFO=%q)", err, zoneinfo)
			}
			return fmt.Errorf("%w (on Windows platforms, you may have to pass the time zone database using the ZONEINFO environment variable, see https://pkg.go.dev/time#LoadLocation for details)", err)
		}
		return err
	}
	*tz = Location{loc}
	return nil
}

func (ir *InclusiveRange) setBegin(n int) {
	ir.Begin = n
}

func (ir *InclusiveRange) setEnd(n int) {
	ir.End = n
}

func (ir *InclusiveRange) memberFromString(in string) (out int, err error) {
	out, err = strconv.Atoi(in)
	if err != nil {
		return -1, err
	}
	return out, nil
}

func (wr *WeekdayRange) memberFromString(in string) (int, error) {
	day, ok := daysOfWeek[in]
	if !ok {
		return -1, fmt.Errorf("%s is not a valid weekday", in)
	}
	return int(day), nil
}

func (mr *MonthRange) memberFromString(in string) (int, error) {
	mon, ok := months[in]
	if !ok {
		return -1, fmt.Errorf("%s is not a valid month", in)
	}
	return int(mon), nil
}

func (wr *WeekdayRange) UnmarshalText(in []byte) error {
	str := string(in)
	if err := stringableRangeFromString(str, wr); err != nil {
		return err
	}
	if wr.Begin > wr.End {
		return errors.New("start day cannot be before end day")
	}
	return nil
}

func (dr *DayOfMonthRange) UnmarshalText(in []byte) error {
	str := string(in)
	if err := stringableRangeFromString(str, dr); err != nil {
		return err
	}
	// Check beginning <= end accounting for negatives day of month indices as well.
	// Months != 31 days can't be addressed here and are clamped, but at least we can catch blatant errors.
	if dr.Begin == 0 || dr.Begin < -31 || dr.Begin > 31 {
		return fmt.Errorf("%d is not a valid day of the month: out of range", dr.Begin)
	}
	if dr.End == 0 || dr.End < -31 || dr.End > 31 {
		return fmt.Errorf("%d is not a valid day of the month: out of range", dr.End)
	}
	// Restricting here prevents errors where begin > end in longer months but not shorter months.
	if dr.Begin < 0 && dr.End > 0 {
		return fmt.Errorf("end day must be negative if start day is negative")
	}
	// Check begin <= end. We can't know this for sure when using negative indices
	// but we can prevent cases where its always invalid (using 28 day minimum length).
	checkBegin := dr.Begin
	checkEnd := dr.End
	if dr.Begin < 0 {
		checkBegin = 28 + dr.Begin
	}
	if dr.End < 0 {
		checkEnd = 28 + dr.End
	}
	if checkBegin > checkEnd {
		return fmt.Errorf("end day %d is always before start day %d", dr.End, dr.Begin)
	}
	return nil
}

func (mr *MonthRange) UnmarshalText(in []byte) error {
	str := string(in)
	if err := stringableRangeFromString(str, mr); err != nil {
		return err
	}
	if mr.Begin > mr.End {
		return fmt.Errorf("end month %d is before start month %d", mr.End, mr.Begin)
	}
	return nil
}

func (yr *YearRange) UnmarshalText(in []byte) error {
	str := string(in)
	if err := stringableRangeFromString(str, yr); err != nil {
		return err
	}
	if yr.Begin > yr.End {
		return fmt.Errorf("end year %d is before start year %d", yr.End, yr.Begin)
	}
	return nil
}

func (tr *TimeRange) setBegin(n int) {
	tr.StartMinute = n
}

func (tr *TimeRange) setEnd(n int) {
	tr.EndMinute = n
}

func (tr *TimeRange) memberFromString(in string) (int, error) {
	start, err := parseTime(in)
	if err != nil {
		return -1, fmt.Errorf("%s is not a valid time", in)
	}
	return start, nil
}

func (tr *TimeRange) UnmarshalText(in []byte) error {
	str := string(in)
	if err := stringableRangeFromString(str, tr); err != nil {
		return err
	}
	if tr.StartMinute >= tr.EndMinute {
		return errors.New("start time cannot be equal or greater than end time")
	}
	return nil
}

// ContainsTime returns true if the TimeInterval contains the given time, otherwise returns false.
func (tp TimeInterval) ContainsTime(t time.Time) bool {
	if tp.Location != nil {
		t = t.In(tp.Location.Location)
	}
	if tp.Times != nil {
		in := false
		for _, validMinutes := range tp.Times {
			if (t.Hour()*60+t.Minute()) >= validMinutes.StartMinute && (t.Hour()*60+t.Minute()) < validMinutes.EndMinute {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	if tp.DaysOfMonth != nil {
		in := false
		for _, validDates := range tp.DaysOfMonth {
			var begin, end int
			daysInMonth := daysInMonth(t)
			if validDates.Begin < 0 {
				begin = daysInMonth + validDates.Begin + 1
			} else {
				begin = validDates.Begin
			}
			if validDates.End < 0 {
				end = daysInMonth + validDates.End + 1
			} else {
				end = validDates.End
			}
			// Skip clamping if the beginning date is after the end of the month.
			if begin > daysInMonth {
				continue
			}
			// Clamp to the boundaries of the month to prevent crossing into other months.
			begin = clamp(begin, -1*daysInMonth, daysInMonth)
			end = clamp(end, -1*daysInMonth, daysInMonth)
			if t.Day() >= begin && t.Day() <= end {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	if tp.Months != nil {
		in := false
		for _, validMonths := range tp.Months {
			if t.Month() >= time.Month(validMonths.Begin) && t.Month() <= time.Month(validMonths.End) {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	if tp.Weekdays != nil {
		in := false
		for _, validDays := range tp.Weekdays {
			if t.Weekday() >= time.Weekday(validDays.Begin) && t.Weekday() <= time.Weekday(validDays.End) {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	if tp.Years != nil {
		in := false
		for _, validYears := range tp.Years {
			if t.Year() >= validYears.Begin && t.Year() <= validYears.End {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	return true
}

var (
	validTime   = "^((([01][0-9])|(2[0-3])):[0-5][0-9])$|(^24:00$)"
	validTimeRE = regexp.MustCompile(validTime)
)

// Given a time, determines the number of days in the month that time occurs in.
func daysInMonth(t time.Time) int {
	monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)
	diff := monthEnd.Sub(monthStart)
	return int(diff.Hours() / 24)
}

func clamp(n, min, max int) int {
	if n <= min {
		return min
	}
	if n >= max {
		return max
	}
	return n
}

// Converts a string of the form "HH:MM" into the number of minutes elapsed in the day.
func parseTime(in string) (mins int, err error) {
	if !validTimeRE.MatchString(in) {
		return 0, fmt.Errorf("couldn't parse timestamp %s, invalid format", in)
	}
	timestampComponents := strings.Split(in, ":")
	if len(timestampComponents) != 2 {
		return 0, fmt.Errorf("invalid timestamp format: %s", in)
	}
	timeStampHours, err := strconv.Atoi(timestampComponents[0])
	if err != nil {
		return 0, err
	}
	timeStampMinutes, err := strconv.Atoi(timestampComponents[1])
	if err != nil {
		return 0, err
	}
	if timeStampHours < 0 || timeStampHours > 24 || timeStampMinutes < 0 || timeStampMinutes > 60 {
		return 0, fmt.Errorf("timestamp %s out of range", in)
	}
	// Timestamps are stored as minutes elapsed in the day, so multiply hours by 60.
	mins = timeStampHours*60 + timeStampMinutes
	return mins, nil
}

// Converts a range that can be represented as strings (e.g. monday:wednesday) into an equivalent integer-represented range.
func stringableRangeFromString(in string, r stringableRange) (err error) {
	in = strings.ToLower(in)
	if strings.ContainsRune(in, '~') {
		components := strings.Split(in, "~")
		if len(components) != 2 {
			return fmt.Errorf("couldn't parse range %s, invalid format", in)
		}
		start, err := r.memberFromString(title.String(components[0]))
		if err != nil {
			return err
		}
		End, err := r.memberFromString(title.String(components[1]))
		if err != nil {
			return err
		}
		r.setBegin(start)
		r.setEnd(End)
		return nil
	}
	val, err := r.memberFromString(title.String(in))
	if err != nil {
		return err
	}
	r.setBegin(val)
	r.setEnd(val)
	return nil
}
