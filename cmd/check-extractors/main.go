// Command check-extractors hits every registered store extractor with a live
// search and prints a working / empty / error / timeout table.
//
//	go run ./cmd/check-extractors
//	go run ./cmd/check-extractors -id boots_uk_v1
//	go run ./cmd/check-extractors -q "olaplex no 3"
//	go run ./cmd/check-extractors -json results.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"muambr-api/extractors"
	"muambr-api/handlers"
	"muambr-api/models"
	"muambr-api/utils"
)

type status string

const (
	statusOK      status = "ok"
	statusEmpty   status = "empty"
	statusError   status = "error"
	statusTimeout status = "timeout"
)

type result struct {
	id          string
	country     string
	category    string
	query       string
	status      status
	count       int
	rawCount    int
	sample      string
	err         string
	duration    time.Duration
	comparisons []models.ProductComparison
}

func main() {
	queryFlag := flag.String("q", "", "search query for every extractor (default: category-specific)")
	idFlag := flag.String("id", "", "run only this extractor identifier")
	timeoutFlag := flag.Duration("timeout", 30*time.Second, "per-extractor timeout")
	parallelFlag := flag.Int("parallel", 4, "max concurrent extractors")
	verboseFlag := flag.Bool("v", false, "print extractor errors")
	jsonFlag := flag.String("json", "", "write full results as JSON to this path")
	flag.Parse()

	utils.InitNopLogger()

	handler := handlers.NewExtractorHandler()
	list := handler.Extractors()
	sort.Slice(list, func(i, j int) bool {
		return list[i].GetIdentifier() < list[j].GetIdentifier()
	})

	if *idFlag != "" {
		filtered := list[:0]
		for _, e := range list {
			if e.GetIdentifier() == *idFlag {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "unknown extractor %q\n", *idFlag)
			os.Exit(2)
		}
		list = filtered
	}

	if *parallelFlag < 1 {
		*parallelFlag = 1
	}

	fmt.Printf("Checking %d extractor(s) (timeout %s, parallel %d)\n\n", len(list), *timeoutFlag, *parallelFlag)

	out := make([]result, len(list))
	sem := make(chan struct{}, *parallelFlag)
	var wg sync.WaitGroup
	for i, e := range list {
		wg.Add(1)
		go func(i int, e extractors.Extractor) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = runOne(e, *queryFlag, *timeoutFlag)
		}(i, e)
	}
	wg.Wait()

	printTable(out, *verboseFlag)
	if *jsonFlag != "" {
		if err := writeJSON(*jsonFlag, out); err != nil {
			fmt.Fprintf(os.Stderr, "write json: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("\nwrote %s\n", *jsonFlag)
	}
	if unhealthy(out) > 0 {
		os.Exit(1)
	}
}

func runOne(e extractors.Extractor, queryOverride string, timeout time.Duration) result {
	id := e.GetIdentifier()
	r := result{
		id:       id,
		country:  string(e.GetCountryCode()),
		category: string(e.GetCategory()),
		query:    queryFor(e, queryOverride),
	}
	if id == "acharpromo_v2" && timeout < 90*time.Second {
		timeout = 90 * time.Second
	}

	done := make(chan result, 1)
	start := time.Now()
	go func() {
		comparisons, err := e.GetComparisons(r.query)
		got := result{
			id:       r.id,
			country:  r.country,
			category: r.category,
			query:    r.query,
			duration: time.Since(start),
		}
		if err != nil {
			got.status = statusError
			got.err = err.Error()
			done <- got
			return
		}
		if comparisons == nil {
			comparisons = []models.ProductComparison{}
		}
		priced := 0
		var sample string
		for i := range comparisons {
			conf := utils.MatchConfidence(r.query, comparisons[i].ProductName)
			comparisons[i].MatchConfidence = &conf
			c := comparisons[i]
			if strings.TrimSpace(c.ProductName) == "" || c.Price <= 0 {
				continue
			}
			priced++
			if sample == "" {
				sample = fmt.Sprintf("%s  %.2f %s", truncate(c.ProductName, 48), c.Price, c.Currency)
			}
		}
		got.comparisons = comparisons
		got.rawCount = len(comparisons)
		got.count = priced
		got.sample = sample
		if priced == 0 {
			got.status = statusEmpty
		} else {
			got.status = statusOK
		}
		done <- got
	}()

	select {
	case got := <-done:
		return got
	case <-time.After(timeout):
		r.status = statusTimeout
		r.duration = timeout
		return r
	}
}

func queryFor(e extractors.Extractor, override string) string {
	if override != "" {
		return override
	}
	switch e.GetIdentifier() {
	case "boozyshop_nl_v1":
		return "loreal"
	case "auchan_pt_v1":
		return "leite"
	case "americanas_br_v1":
		return "smart tv lg"
	case "zoom_br_v1":
		return "geladeira electrolux"
	case "carrefour_br_v1", "fastshop_br_v1":
		return "air fryer"
	}
	switch e.GetCategory() {
	case models.CategoryBeauty:
		return "armani code"
	case models.CategoryAppliances:
		return "air fryer"
	default:
		return "iphone"
	}
}

type resultJSON struct {
	ID          string                     `json:"id"`
	Country     string                     `json:"country"`
	Category    string                     `json:"category"`
	Query       string                     `json:"query"`
	Status      string                     `json:"status"`
	Count       int                        `json:"count"`
	RawCount    int                        `json:"raw_count"`
	Sample      string                     `json:"sample,omitempty"`
	Error       string                     `json:"error,omitempty"`
	DurationMS  int64                      `json:"duration_ms"`
	Comparisons []models.ProductComparison `json:"comparisons"`
}

type reportJSON struct {
	GeneratedAt string       `json:"generated_at"`
	Working     int          `json:"working"`
	Empty       int          `json:"empty"`
	Error       int          `json:"error"`
	Timeout     int          `json:"timeout"`
	Total       int          `json:"total"`
	NeedsFix    []string     `json:"needs_fix"`
	Results     []resultJSON `json:"results"`
}

func writeJSON(path string, rows []result) error {
	ok, empty, errored, timed := 0, 0, 0, 0
	needsFix := make([]string, 0)
	out := make([]resultJSON, 0, len(rows))
	for _, r := range rows {
		switch r.status {
		case statusOK:
			ok++
		case statusEmpty:
			empty++
			needsFix = append(needsFix, r.id)
		case statusError:
			errored++
			needsFix = append(needsFix, r.id)
		case statusTimeout:
			timed++
			needsFix = append(needsFix, r.id)
		}
		comparisons := r.comparisons
		if comparisons == nil {
			comparisons = []models.ProductComparison{}
		}
		out = append(out, resultJSON{
			ID:          r.id,
			Country:     r.country,
			Category:    r.category,
			Query:       r.query,
			Status:      string(r.status),
			Count:       r.count,
			RawCount:    r.rawCount,
			Sample:      r.sample,
			Error:       r.err,
			DurationMS:  r.duration.Milliseconds(),
			Comparisons: comparisons,
		})
	}
	payload := reportJSON{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Working:     ok,
		Empty:       empty,
		Error:       errored,
		Timeout:     timed,
		Total:       len(rows),
		NeedsFix:    needsFix,
		Results:     out,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func printTable(rows []result, verbose bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tID\tCOUNTRY\tCATEGORY\tN\tTIME\tQUERY")
	ok, empty, errored, timed := 0, 0, 0, 0
	for _, r := range rows {
		switch r.status {
		case statusOK:
			ok++
		case statusEmpty:
			empty++
		case statusError:
			errored++
		case statusTimeout:
			timed++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			r.status, r.id, r.country, r.category, r.count, r.duration.Truncate(time.Millisecond), r.query)
	}
	_ = w.Flush()

	fmt.Printf("\nworking %d  empty %d  error %d  timeout %d  total %d\n",
		ok, empty, errored, timed, len(rows))

	needsFix := make([]result, 0)
	for _, r := range rows {
		if r.status != statusOK {
			needsFix = append(needsFix, r)
		}
	}
	if len(needsFix) == 0 {
		fmt.Println("all extractors returned priced offers")
		return
	}

	fmt.Println("\nneeds fix:")
	for _, r := range needsFix {
		detail := r.err
		if detail == "" {
			detail = "no priced offers"
		}
		fmt.Printf("  - %s (%s/%s): %s %s\n", r.id, r.country, r.category, r.status, truncate(detail, 120))
	}

	if verbose {
		fmt.Println("\nsamples:")
		for _, r := range rows {
			if r.sample != "" {
				fmt.Printf("  %s  %s\n", r.id, r.sample)
			}
		}
	}
}

func unhealthy(rows []result) int {
	n := 0
	for _, r := range rows {
		if r.status != statusOK {
			n++
		}
	}
	return n
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
