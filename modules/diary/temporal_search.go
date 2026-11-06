package diary

import (
	"regexp"
	"strings"
	"time"
	"fmt"
)

type TemporalQuery struct {
	Start   time.Time
	End     time.Time
	Keyword string
}

type SearchableDiaryCard interface {
	GetID() string
	GetCreatedAt() time.Time
	GetObservedHour() string
	GetContent() string
	GetTitle() string
	GetTags() []string
	GetLocationName() string
	GetLocationLandmark() string
}

func ParseTemporalQuery(query string, now time.Time) (*TemporalQuery, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, false
	}

	// Helper to extract keywords
	extractKeywords := func(matchedStr string) string {
		return strings.TrimSpace(strings.Replace(query, matchedStr, "", 1))
	}

	// Range matches
	rangeRegexes := []string{
		`^(\d{4}[-/]\d{2}[-/]\d{2}(?:\s+\d{1,2}(?::\d{2})?)?)\s*(?:~|to)\s*(\d{4}[-/]\d{2}[-/]\d{2}(?:\s+\d{1,2}(?::\d{2})?)?)`,
		`^(今天|昨天|前天)(?:\s+(\d{1,2}(?::\d{2})?))?\s*(?:~|to)\s*(今天|昨天|前天)(?:\s+(\d{1,2}(?::\d{2})?))?`,
		`^(\d{4}[-/]\d{2}[-/]\d{2})\s+(\d{1,2}(?::\d{2})?)\s*(?:~|to)\s*(\d{1,2}(?::\d{2})?)`,
	}

	for i, r := range rangeRegexes {
		re := regexp.MustCompile(r)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 0 {
			var start, end time.Time
			var err1, err2 error

			if i == 0 {
				start, err1 = parseTimeSegment(matches[1], now)
				end, err2 = parseTimeSegment(matches[2], now)
				if err1 == nil && err2 == nil {
					// If end has no time, make it end of day
					if len(matches[2]) <= 10 {
						end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, end.Location())
					} else {
						// If end has time but no minutes, make it end of hour
						if !strings.Contains(matches[2][11:], ":") {
							end = time.Date(end.Year(), end.Month(), end.Day(), end.Hour(), 59, 59, 999999999, end.Location())
						}
					}

					return &TemporalQuery{
						Start:   start,
						End:     end,
						Keyword: extractKeywords(matches[0]),
					}, true
				}
			} else if i == 1 {
				sDate := parseRelativeDay(matches[1], now)
				eDate := parseRelativeDay(matches[3], now)
				sTimeStr := matches[2]
				eTimeStr := matches[4]

				start = sDate
				if sTimeStr != "" {
					h, m := parseHourMinute(sTimeStr)
					start = time.Date(sDate.Year(), sDate.Month(), sDate.Day(), h, m, 0, 0, sDate.Location())
				}

				end = eDate
				if eTimeStr != "" {
					h, m := parseHourMinute(eTimeStr)
					end = time.Date(eDate.Year(), eDate.Month(), eDate.Day(), h, m, 59, 999999999, eDate.Location())
				} else {
					end = time.Date(eDate.Year(), eDate.Month(), eDate.Day(), 23, 59, 59, 999999999, eDate.Location())
				}

				return &TemporalQuery{
					Start:   start,
					End:     end,
					Keyword: extractKeywords(matches[0]),
				}, true
			} else if i == 2 {
				date, _ := parseTimeSegment(matches[1], now)
				sh, sm := parseHourMinute(matches[2])
				eh, em := parseHourMinute(matches[3])

				start = time.Date(date.Year(), date.Month(), date.Day(), sh, sm, 0, 0, date.Location())
				end = time.Date(date.Year(), date.Month(), date.Day(), eh, em, 59, 999999999, date.Location())

				return &TemporalQuery{
					Start:   start,
					End:     end,
					Keyword: extractKeywords(matches[0]),
				}, true
			}
		}
	}

	// Exact Hour matches
	hourRegexes := []string{
		`^(\d{4}[-/]\d{2}[-/]\d{2})[T\s](\d{1,2})(?:[:\s]00|h|点)?(?:\s|$)`,
		`^(\d{2}-\d{2})\s+(\d{1,2})(?:[:\s]00|h|点)?(?:\s|$)`,
		`^(今天|昨天|前天)\s+(\d{1,2})(?:[:\s]00|h|点)?(?:\s|$)`,
	}

	for i, r := range hourRegexes {
		re := regexp.MustCompile(r)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 0 {
			var date time.Time
			var h int
			if i == 0 {
				date, _ = parseTimeSegment(matches[1], now)
				h, _ = parseHourMinute(matches[2])
			} else if i == 1 {
				date, _ = parseTimeSegment(now.Format("2006")+"-"+matches[1], now)
				h, _ = parseHourMinute(matches[2])
			} else if i == 2 {
				date = parseRelativeDay(matches[1], now)
				h, _ = parseHourMinute(matches[2])
			}

			start := time.Date(date.Year(), date.Month(), date.Day(), h, 0, 0, 0, date.Location())
			end := time.Date(date.Year(), date.Month(), date.Day(), h, 59, 59, 999999999, date.Location())

			loc := re.FindStringIndex(query)
			kw := ""
			if loc != nil {
				kw = strings.TrimSpace(query[loc[1]:])
			}

			return &TemporalQuery{
				Start:   start,
				End:     end,
				Keyword: kw,
			}, true
		}
	}

	// Single Day / Month
	dayMonthRegexes := []string{
		`^(\d{4}[-/]\d{2}[-/]\d{2})(?:\s|$)`,
		`^(\d{4}[-/]\d{2})(?:\s|$)`,
		`^(今天|昨天|前天)(?:\s|$)`,
	}

	for i, r := range dayMonthRegexes {
		re := regexp.MustCompile(r)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 0 {
			var start, end time.Time
			if i == 0 {
				date, _ := parseTimeSegment(matches[1], now)
				start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
				end = time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
			} else if i == 1 {
				t, _ := time.ParseInLocation("2006-01", strings.ReplaceAll(matches[1], "/", "-"), now.Location())
				start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
				end = start.AddDate(0, 1, -1)
				end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, end.Location())
			} else if i == 2 {
				date := parseRelativeDay(matches[1], now)
				start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
				end = time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
			}

			loc := re.FindStringIndex(query)
			kw := ""
			if loc != nil {
				kw = strings.TrimSpace(query[loc[1]:])
			}

			return &TemporalQuery{
				Start:   start,
				End:     end,
				Keyword: kw,
			}, true
		}
	}

	return nil, false
}

func parseTimeSegment(s string, now time.Time) (time.Time, error) {
	s = strings.ReplaceAll(s, "/", "-")
	if len(s) == 10 {
		return time.ParseInLocation("2006-01-02", s, now.Location())
	} else if len(s) >= 13 {
		if strings.Contains(s, ":") {
			return time.ParseInLocation("2006-01-02 15:04", s[:16], now.Location())
		} else {
			return time.ParseInLocation("2006-01-02 15", s[:13], now.Location())
		}
	}
	return time.ParseInLocation("2006-01-02", s, now.Location())
}

func parseRelativeDay(s string, now time.Time) time.Time {
	switch s {
	case "今天":
		return now
	case "昨天":
		return now.AddDate(0, 0, -1)
	case "前天":
		return now.AddDate(0, 0, -2)
	}
	return now
}

func parseHourMinute(s string) (int, int) {
	parts := strings.Split(strings.Trim(s, "h点"), ":")
	h := 0
	m := 0
	if len(parts) > 0 {
		fmt.Sscanf(parts[0], "%d", &h)
	}
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &m)
	}
	return h, m
}

func FilterDiaryCards(cards []SearchableDiaryCard, query string, now time.Time) []string {
	tq, ok := ParseTemporalQuery(query, now)
	if !ok {
		return nil
	}

	var matchedIDs []string
	for _, c := range cards {
		cardTime := c.GetCreatedAt()

		if !tq.Start.IsZero() && !tq.End.IsZero() {
			if cardTime.Before(tq.Start) || cardTime.After(tq.End) {
				obsHour := c.GetObservedHour()
				if obsHour != "" {
					obsTime, err := time.Parse("2006-01-02 15", obsHour)
					if err != nil || obsTime.Before(tq.Start) || obsTime.After(tq.End) {
						continue
					}
				} else {
					continue
				}
			}
		}

		if tq.Keyword != "" {
			kw := strings.ToLower(tq.Keyword)
			match := false
			if strings.Contains(strings.ToLower(c.GetTitle()), kw) ||
				strings.Contains(strings.ToLower(c.GetContent()), kw) ||
				strings.Contains(strings.ToLower(c.GetLocationName()), kw) ||
				strings.Contains(strings.ToLower(c.GetLocationLandmark()), kw) {
				match = true
			}
			if !match {
				for _, t := range c.GetTags() {
					if strings.Contains(strings.ToLower(t), kw) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		matchedIDs = append(matchedIDs, c.GetID())
	}
	return matchedIDs
}
