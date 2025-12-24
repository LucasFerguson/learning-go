package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TakeoutActivity struct {
	Title     string `json:"title"`
	TitleURL  string `json:"titleUrl"`
	Time      string `json:"time"`
	Subtitles []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"subtitles"`
}

type ChannelStat struct {
	ChannelName string `json:"channel_name"`
	ChannelURL  string `json:"channel_url,omitempty"`
	WatchCount  int    `json:"watch_count"`
}

type YearResult struct {
	Year             int           `json:"year"`
	TotalVideos      int           `json:"total_videos_watched"`
	UniqueChannels   int           `json:"unique_channels"`
	TopChannels      []ChannelStat `json:"top_channels"`
	TopN             int           `json:"top_n"`
	FilteredAction   string        `json:"filtered_action"`
	TimeParseFailures int          `json:"time_parse_failures"`
}

type Summary struct {
	YearRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"year_range"`
	TotalVideosAllYears int                 `json:"total_videos_all_years"`
	Years               map[int]YearResult  `json:"years"`
}

type channelKey struct {
	name string
	url  string
}

func main() {
	inPath := flag.String("in", "", "Path to watch-history.json (required)")
	outDir := flag.String("outdir", "out", "Output directory to write JSON files into")
	startYear := flag.Int("start", 2020, "Start year (inclusive)")
	endYear := flag.Int("end", 2026, "End year (inclusive)")
	topN := flag.Int("top", 6, "Top N channels per year")
	fullLimit := flag.Int("full-limit", 0, "Limit for channels_full_<YEAR>.json (0 = all channels)")
	allTimeTop := flag.Int("alltime-top", 100, "Top N channels for all-time output")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "error: -in is required")
		os.Exit(2)
	}
	if *startYear > *endYear {
		fmt.Fprintln(os.Stderr, "error: -start must be <= -end")
		os.Exit(2)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating outdir:", err)
		os.Exit(1)
	}

	f, err := os.Open(*inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening input:", err)
		os.Exit(1)
	}
	defer f.Close()

	yearCounts := make(map[int]map[channelKey]int)
	yearTotals := make(map[int]int)
	yearParseFails := make(map[int]int)
	allTimeCounts := make(map[channelKey]int)
	totalAllYears := 0

	// init year buckets
	for y := *startYear; y <= *endYear; y++ {
		yearCounts[y] = make(map[channelKey]int)
		yearTotals[y] = 0
		yearParseFails[y] = 0
	}

	if err := streamParseAndAggregate(f, *startYear, *endYear, yearCounts, yearTotals, yearParseFails, allTimeCounts, &totalAllYears); err != nil {
		fmt.Fprintln(os.Stderr, "error parsing json:", err)
		os.Exit(1)
	}

	// Build per-year results
	perYearTop := make(map[int]YearResult)
	for y := *startYear; y <= *endYear; y++ {
		fullStats := statsFromMap(yearCounts[y])
		sortStatsByCountThenName(fullStats)

		top := fullStats
		if *topN > 0 && len(top) > *topN {
			top = top[:*topN]
		}

		perYearTop[y] = YearResult{
			Year:             y,
			TotalVideos:      yearTotals[y],
			UniqueChannels:   len(yearCounts[y]),
			TopChannels:      top,
			TopN:             *topN,
			FilteredAction:   "Watched",
			TimeParseFailures: yearParseFails[y],
		}

		// Write per-year top file
		if err := writeJSON(filepath.Join(*outDir, fmt.Sprintf("top_channels_%d.json", y)), perYearTop[y]); err != nil {
			fmt.Fprintln(os.Stderr, "error writing year top:", err)
			os.Exit(1)
		}

		// Write per-year full file
		fullOut := fullStats
		if *fullLimit > 0 && len(fullOut) > *fullLimit {
			fullOut = fullOut[:*fullLimit]
		}
		fullPayload := struct {
			Year        int           `json:"year"`
			TotalVideos int           `json:"total_videos_watched"`
			Channels    []ChannelStat `json:"channels_sorted"`
			Limit       int           `json:"limit"`
			Sort        string        `json:"sort"`
		}{
			Year:        y,
			TotalVideos: yearTotals[y],
			Channels:    fullOut,
			Limit:       *fullLimit,
			Sort:        "watch_count desc, channel_name asc",
		}

		if err := writeJSON(filepath.Join(*outDir, fmt.Sprintf("channels_full_%d.json", y)), fullPayload); err != nil {
			fmt.Fprintln(os.Stderr, "error writing year full:", err)
			os.Exit(1)
		}
	}

	// Write combined “top by year” file
	topByYearPayload := struct {
		StartYear int                    `json:"start_year"`
		EndYear   int                    `json:"end_year"`
		TopN      int                    `json:"top_n"`
		Years     map[int]YearResult     `json:"years"`
	}{
		StartYear: *startYear,
		EndYear:   *endYear,
		TopN:      *topN,
		Years:     perYearTop,
	}
	if err := writeJSON(filepath.Join(*outDir, "top_channels_by_year.json"), topByYearPayload); err != nil {
		fmt.Fprintln(os.Stderr, "error writing top_channels_by_year.json:", err)
		os.Exit(1)
	}

	// Write summary file
	var summary Summary
	summary.YearRange.Start = *startYear
	summary.YearRange.End = *endYear
	summary.TotalVideosAllYears = totalAllYears
	summary.Years = perYearTop

	if err := writeJSON(filepath.Join(*outDir, "summary.json"), summary); err != nil {
		fmt.Fprintln(os.Stderr, "error writing summary.json:", err)
		os.Exit(1)
	}

	// Write all-time top channels
	allTimeStats := statsFromMap(allTimeCounts)
	sortStatsByCountThenName(allTimeStats)
	if *allTimeTop > 0 && len(allTimeStats) > *allTimeTop {
		allTimeStats = allTimeStats[:*allTimeTop]
	}
	allTimePayload := struct {
		TopN        int           `json:"top_n"`
		TotalVideos int           `json:"total_videos_counted"`
		Channels    []ChannelStat `json:"channels"`
		Sort        string        `json:"sort"`
		Notes       string        `json:"notes"`
	}{
		TopN:        *allTimeTop,
		TotalVideos: totalAllYears,
		Channels:    allTimeStats,
		Sort:        "watch_count desc, channel_name asc",
		Notes:       "Counts are derived from entries whose title starts with 'Watched ' and whose time parses as RFC3339; however, entries with missing channel info are grouped under '(unknown channel)'.",
	}
	if err := writeJSON(filepath.Join(*outDir, "top_channels_all_time.json"), allTimePayload); err != nil {
		fmt.Fprintln(os.Stderr, "error writing top_channels_all_time.json:", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote JSON outputs to: %s\n", *outDir)
}

func streamParseAndAggregate(
	f *os.File,
	startYear int,
	endYear int,
	yearCounts map[int]map[channelKey]int,
	yearTotals map[int]int,
	yearParseFails map[int]int,
	allTimeCounts map[channelKey]int,
	totalAllYears *int,
) error {
	br := bufio.NewReaderSize(f, 1024*1024)
	dec := json.NewDecoder(br)

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("expected top-level JSON array")
	}

	for dec.More() {
		var a TakeoutActivity
		if err := dec.Decode(&a); err != nil {
			return err
		}

		// Only keep watch events
		title := strings.TrimSpace(a.Title)
		if !strings.HasPrefix(strings.ToLower(title), "watched ") {
			continue
		}

		t, err := time.Parse(time.RFC3339, strings.TrimSpace(a.Time))
		if err != nil {
			// If time is unparseable, we cannot bucket it by year reliably.
			// Still track it as a parse failure for all buckets? We do not know year, so skip.
			continue
		}

		y := t.Year()
		if y < startYear || y > endYear {
			continue
		}

		chName, chURL := extractChannel(a)
		if chName == "" {
			chName = "(unknown channel)"
		}

		k := channelKey{name: chName, url: chURL}
		yearCounts[y][k]++
		yearTotals[y]++
		allTimeCounts[k]++
		*totalAllYears++
	}

	_, _ = dec.Token()
	_ = yearParseFails // kept for future extension if you decide to track per-year parse failures differently
	return nil
}

func extractChannel(a TakeoutActivity) (name, url string) {
	if len(a.Subtitles) == 0 {
		return "", ""
	}
	n := strings.TrimSpace(a.Subtitles[0].Name)
	u := strings.TrimSpace(a.Subtitles[0].URL)
	return n, u
}

func statsFromMap(m map[channelKey]int) []ChannelStat {
	out := make([]ChannelStat, 0, len(m))
	for k, c := range m {
		out = append(out, ChannelStat{
			ChannelName: k.name,
			ChannelURL:  k.url,
			WatchCount:  c,
		})
	}
	return out
}

func sortStatsByCountThenName(stats []ChannelStat) {
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].WatchCount == stats[j].WatchCount {
			return strings.ToLower(stats[i].ChannelName) < strings.ToLower(stats[j].ChannelName)
		}
		return stats[i].WatchCount > stats[j].WatchCount
	})
}

func writeJSON(path string, v any) error {
	tmp := path + ".tmp"

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, path)
}
