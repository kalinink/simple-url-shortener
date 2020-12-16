package shortener

import "time"

type URL struct {
	Long  string
	Short string
}

type NewURL struct {
	Long      string
	Short     string
	CreatedAt time.Time
}

type ShortURL struct {
	URL        string
	AccessTime time.Time
}

type OverallStatistics struct {
	LongURL  Statistics
	ShortURL Statistics
}

type Statistics struct {
	Count  int
	Timing time.Time
}
