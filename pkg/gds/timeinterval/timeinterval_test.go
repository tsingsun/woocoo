package timeinterval

import (
	"github.com/stretchr/testify/assert"
	"github.com/tsingsun/woocoo/pkg/conf"
	"testing"
	"time"
)

func TestTimeIntervalUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		cfgStr   string
		contains []string
		excludes []string
		want     []TimeInterval
		wantErr  bool
		err      string
	}{
		{
			name: "Simple business hours test",
			cfgStr: `
ts:
- weekdays: ['Monday~Friday']
  times: ['09:00~17:00']
`,
			want: []TimeInterval{
				{
					Weekdays: []WeekdayRange{{InclusiveRange{Begin: 1, End: 5}}},
					Times:    []TimeRange{{StartMinute: 540, EndMinute: 1020}},
				},
			},
			contains: []string{
				"08 Jul 20 09:00 +0000",
				"08 Jul 20 16:59 +0000",
			},
			excludes: []string{
				"08 Jul 20 05:00 +0000",
				"08 Jul 20 08:59 +0000",
			},
		},
		{
			name: "More advanced test with negative indices and ranges",
			cfgStr: `
# Last week, excluding Saturday, of the first quarter of the year during business hours from 2020 to 2025 and 2030-2035
ts:
- weekdays: ['monday~friday', 'sunday']
  months: ['january~march']
  daysOfMonth: ['-7~-1']
  years: ['2020~2025', '2030~2035']
  times: ['09:00~17:00']
`,
			want: []TimeInterval{
				{
					Weekdays:    []WeekdayRange{{InclusiveRange{Begin: 1, End: 5}}, {InclusiveRange{Begin: 0, End: 0}}},
					Times:       []TimeRange{{StartMinute: 540, EndMinute: 1020}},
					Months:      []MonthRange{{InclusiveRange{1, 3}}},
					DaysOfMonth: []DayOfMonthRange{{InclusiveRange{-7, -1}}},
					Years:       []YearRange{{InclusiveRange{2020, 2025}}, {InclusiveRange{2030, 2035}}},
				},
			},
			contains: []string{
				"27 Jan 21 09:00 +0000",
				"28 Jan 21 16:59 +0000",
				"29 Jan 21 13:00 +0000",
				"31 Mar 25 13:00 +0000",
				"31 Mar 25 13:00 +0000",
				"31 Jan 35 13:00 +0000",
			},
			excludes: []string{
				"30 Jan 21 13:00 +0000", // Saturday
				"01 Apr 21 13:00 +0000", // 4th month
				"30 Jan 26 13:00 +0000", // 2026
				"31 Jan 35 17:01 +0000", // After 5pm
			},
			wantErr: false,
		},
		{
			name: "Invalid start time",
			cfgStr: `
ts:
- times: ["01:99~23:59"]
`,
			wantErr: true,
			err:     "01:99 is not a valid time",
		},
		{
			name: "Invalid end time",
			cfgStr: `
ts:
- times: ["00:00~99:99"]
`,
			wantErr: true,
			err:     "99:99 is not a valid time",
		},
		{
			name: "Start day before end day",
			cfgStr: `
ts:
- weekdays: ['friday~monday']
`,
			wantErr: true,
			err:     "start day cannot be before end day",
		},
		{
			name: "Invalid weekdays",
			cfgStr: `
ts:
- weekdays: ['blurgsday~flurgsday']
`,
			wantErr: true,
			err:     "is not a valid weekday",
		},
		{
			name: "Numeric weekdays aren't allowed",
			cfgStr: `
ts:
- weekdays: ['1~3']
`,
			wantErr: true,
			err:     "is not a valid weekday",
		},
		{
			name: "Negative numeric weekdays aren't allowed",
			cfgStr: `
ts:
- weekdays: ['-2~-1']
`,
			wantErr: true,
			err:     "is not a valid weekday",
		},
		{
			name: "0 day of month",
			cfgStr: `
ts:
- daysOfMonth: ['0']
`,
			wantErr: true,
			err:     "0 is not a valid day of the month: out of range",
		},
		{
			name: "Start day of month < 0",
			cfgStr: `
ts:
- daysOfMonth: ['-50~-20']
`,
			wantErr: true,
			err:     "-50 is not a valid day of the month: out of range",
		},
		{
			name: "End day of month > 31",
			cfgStr: `
ts:
- daysOfMonth: ['1~50']
`,
			wantErr: true,
			err:     "50 is not a valid day of the month: out of range",
		},
		{
			name: "Negative indices should work",
			cfgStr: `
ts:
- daysOfMonth: ['1~-1']
`,
			want: []TimeInterval{
				{
					DaysOfMonth: []DayOfMonthRange{{InclusiveRange{1, -1}}},
				},
			},
			wantErr: false,
		},
		{
			name: "End day must be negative if begin day is negative",
			cfgStr: `
ts:
- daysOfMonth: ['-15~5']
`,
			wantErr: true,
			err:     "end day must be negative if start day is negative",
		},
		{
			name: "Negative end date before positive positive start date",
			cfgStr: `
ts:
- daysOfMonth: ['10~-25']
`,
			wantErr: true,
			err:     "end day -25 is always before start day 10",
		},
		{
			name: "Months should work regardless of case",
			cfgStr: `
ts:
- months: ['January~december']
`,
			want: []TimeInterval{
				{
					Months: []MonthRange{{InclusiveRange{1, 12}}},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid start month",
			cfgStr: `
ts:
- months: ['martius~december']
`,
			wantErr: true,
			err:     "is not a valid month",
		},
		{
			name: "Invalid end month",
			cfgStr: `
ts:
- months: ['January~martius']
`,
			wantErr: true,
			err:     "is not a valid month",
		},
		{
			name: "Start month after end month",
			cfgStr: `
ts:
- months: ['december~january']
`,
			wantErr: true,
			err:     "end month 1 is before start month 12",
		},
		{
			name: "Time zones may be specified by location",
			cfgStr: `
ts:
- years: ['2020~2022']
  location: 'Asia/Shanghai'
`,
			want: []TimeInterval{
				{
					Years:    []YearRange{{InclusiveRange{2020, 2022}}},
					Location: &Location{mustLoadLocation("Asia/Shanghai")},
				},
			},
			wantErr: false,
		},
		{
			name: "Start year after end year",
			cfgStr: `
ts:
- years: ['2022~2020']
`,
			wantErr: true,
			err:     "end year 2020 is before start year 2022",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ti struct {
				Ts []TimeInterval //nolint:stylecheck
			}
			cfg := conf.NewFromBytes([]byte(tt.cfgStr))
			err := cfg.Unmarshal(&ti)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.err)
				return
			}
			assert.Equal(t, tt.want, ti.Ts)
			for _, ts := range tt.contains {
				_t, _ := time.Parse(time.RFC822Z, ts)
				isContained := false
				for _, interval := range ti.Ts {
					if interval.ContainsTime(_t) {
						isContained = true
					}
				}
				if !isContained {
					t.Errorf("Expected intervals to contain time %s", _t)
				}
			}
			for _, ts := range tt.excludes {
				_t, _ := time.Parse(time.RFC822Z, ts)
				isContained := false
				for _, interval := range ti.Ts {
					if interval.ContainsTime(_t) {
						isContained = true
					}
				}
				if isContained {
					t.Errorf("Expected intervals to exclude time %s", _t)
				}
			}
		})
	}
}

// Utility function for declaring time locations in test cases. Panic if the location can't be loaded.
func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
