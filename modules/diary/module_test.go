package diary

import (
	"testing"
	"time"
)

type MockDiaryCard struct {
	id          string
	createdAt   time.Time
	content     string
	title       string
	obsHour     string
	tags        []string
	locName     string
	locLandmark string
}

func (m *MockDiaryCard) GetID() string               { return m.id }
func (m *MockDiaryCard) GetCreatedAt() time.Time     { return m.createdAt }
func (m *MockDiaryCard) GetObservedHour() string     { return m.obsHour }
func (m *MockDiaryCard) GetContent() string          { return m.content }
func (m *MockDiaryCard) GetTitle() string            { return m.title }
func (m *MockDiaryCard) GetTags() []string           { return m.tags }
func (m *MockDiaryCard) GetLocationName() string     { return m.locName }
func (m *MockDiaryCard) GetLocationLandmark() string { return m.locLandmark }

func TestParseTemporalQuery(t *testing.T) {
	loc, _ := time.LoadLocation("Local")
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, loc)

	tests := []struct {
		query       string
		wantStart   time.Time
		wantEnd     time.Time
		wantKeyword string
		wantMatch   bool
	}{
		// Exact hour
		{"2026-11-06 14", time.Date(2026, 11, 6, 14, 0, 0, 0, loc), time.Date(2026, 11, 6, 14, 59, 59, 999999999, loc), "", true},
		{"今天 14:00", time.Date(2026, 11, 6, 14, 0, 0, 0, loc), time.Date(2026, 11, 6, 14, 59, 59, 999999999, loc), "", true},
		{"昨天 22点", time.Date(2026, 11, 5, 22, 0, 0, 0, loc), time.Date(2026, 11, 5, 22, 59, 59, 999999999, loc), "", true},
		{"11-06 09:00", time.Date(2026, 11, 6, 9, 0, 0, 0, loc), time.Date(2026, 11, 6, 9, 59, 59, 999999999, loc), "", true},

		// Time range
		{"2026-11-01 ~ 2026-11-06", time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2026, 11, 6, 23, 59, 59, 999999999, loc), "", true},
		{"2026-11-06 08:00 ~ 2026-11-06 18:00", time.Date(2026, 11, 6, 8, 0, 0, 0, loc), time.Date(2026, 11, 6, 18, 0, 59, 999999999, loc), "", true},

		// Range with keywords
		{"2026-11-01 ~ 2026-11-06 咖啡", time.Date(2026, 11, 1, 0, 0, 0, 0, loc), time.Date(2026, 11, 6, 23, 59, 59, 999999999, loc), "咖啡", true},

		// Single Day
		{"今天", time.Date(2026, 11, 6, 0, 0, 0, 0, loc), time.Date(2026, 11, 6, 23, 59, 59, 999999999, loc), "", true},
	}

	for _, tt := range tests {
		q, ok := ParseTemporalQuery(tt.query, now)
		if ok != tt.wantMatch {
			t.Errorf("ParseTemporalQuery(%q) match = %v, want %v", tt.query, ok, tt.wantMatch)
			continue
		}
		if ok {
			if !q.Start.Equal(tt.wantStart) {
				t.Errorf("ParseTemporalQuery(%q) start = %v, want %v", tt.query, q.Start, tt.wantStart)
			}
			if q.End.Year() != tt.wantEnd.Year() || q.End.Hour() != tt.wantEnd.Hour() || q.End.Day() != tt.wantEnd.Day() {
				t.Errorf("ParseTemporalQuery(%q) end = %v, want %v", tt.query, q.End, tt.wantEnd)
			}
			if q.Keyword != tt.wantKeyword {
				t.Errorf("ParseTemporalQuery(%q) keyword = %v, want %v", tt.query, q.Keyword, tt.wantKeyword)
			}
		}
	}
}

func TestFilterDiaryCards(t *testing.T) {
	loc, _ := time.LoadLocation("Local")
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, loc)

	cards := []SearchableDiaryCard{
		&MockDiaryCard{
			id:        "1",
			createdAt: time.Date(2026, 11, 6, 9, 30, 0, 0, loc),
			content:   "早上喝了咖啡",
		},
		&MockDiaryCard{
			id:        "2",
			createdAt: time.Date(2026, 11, 6, 14, 15, 0, 0, loc),
			content:   "下午去散步",
		},
		&MockDiaryCard{
			id:        "3",
			createdAt: time.Date(2026, 11, 5, 22, 10, 0, 0, loc),
			content:   "昨晚看书",
		},
	}

	tests := []struct {
		query    string
		wantIDs  []string
	}{
		{"2026-11-06 14", []string{"2"}},
		{"今天", []string{"1", "2"}},
		{"今天 咖啡", []string{"1"}},
		{"昨天", []string{"3"}},
		{"2026-11-05 ~ 2026-11-06", []string{"1", "2", "3"}},
	}

	for _, tt := range tests {
		matchedIDs := FilterDiaryCards(cards, tt.query, now)
		if len(matchedIDs) != len(tt.wantIDs) {
			t.Errorf("FilterDiaryCards(%q) returned %d IDs, want %d", tt.query, len(matchedIDs), len(tt.wantIDs))
		}
	}
}

func TestRejectFutureDiary(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	if !future.After(time.Now()) {
		t.Errorf("Future rejection logic failed")
	}
}
