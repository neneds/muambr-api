// Command check-linkparsers fetches a product page for each registered link
// parser and reports whether title and price were extracted.
//
//	go run ./cmd/check-linkparsers
//	go run ./cmd/check-linkparsers -host amazon.com.br
//	go run ./cmd/check-linkparsers -url 'https://www.example.com/product'
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"muambr-api/linkparsers"
	"muambr-api/utils"
)

type status string

const (
	statusOK      status = "ok"
	statusNoPrice status = "no_price"
	statusEmpty   status = "empty"
	statusError   status = "error"
	statusTimeout status = "timeout"
	statusNoProbe status = "no_probe"
)

type probe struct {
	host   string
	parser string
	url    string
}

type result struct {
	probe
	status   status
	title    string
	price    string
	currency string
	err      string
	duration time.Duration
}

func main() {
	hostFlag := flag.String("host", "", "check only this registered hostname")
	urlFlag := flag.String("url", "", "check a single product URL (any host)")
	timeoutFlag := flag.Duration("timeout", 25*time.Second, "per-URL timeout")
	parallelFlag := flag.Int("parallel", 3, "max concurrent fetches")
	verboseFlag := flag.Bool("v", false, "print titles and errors")
	flag.Parse()

	utils.InitNopLogger()

	probes := collectProbes(*hostFlag, *urlFlag)
	if len(probes) == 0 {
		fmt.Fprintf(os.Stderr, "no link parsers to check\n")
		os.Exit(2)
	}
	if *parallelFlag < 1 {
		*parallelFlag = 1
	}

	fmt.Printf("Checking %d link parser(s) (timeout %s, parallel %d)\n\n", len(probes), *timeoutFlag, *parallelFlag)

	out := make([]result, len(probes))
	sem := make(chan struct{}, *parallelFlag)
	var wg sync.WaitGroup
	for i, p := range probes {
		wg.Add(1)
		go func(i int, p probe) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = runOne(p, *timeoutFlag)
		}(i, p)
	}
	wg.Wait()

	printTable(out, *verboseFlag)
	if unhealthy(out) > 0 {
		os.Exit(1)
	}
}

func collectProbes(host, rawURL string) []probe {
	if rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" {
			fmt.Fprintf(os.Stderr, "invalid url %q\n", rawURL)
			os.Exit(2)
		}
		h := strings.TrimPrefix(strings.ToLower(parsed.Host), "www.")
		return []probe{{host: h, parser: linkparsers.ParserName(h), url: rawURL}}
	}

	hosts := linkparsers.RegisteredHosts()
	if host != "" {
		found := false
		for _, h := range hosts {
			if h == host {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "unknown host %q\n", host)
			os.Exit(2)
		}
		hosts = []string{host}
	}

	out := make([]probe, 0, len(hosts))
	for _, h := range hosts {
		u, _ := linkparsers.ProbeURL(h)
		out = append(out, probe{host: h, parser: linkparsers.ParserName(h), url: u})
	}
	return out
}

func runOne(p probe, timeout time.Duration) result {
	r := result{probe: p}
	if p.url == "" {
		r.status = statusNoProbe
		return r
	}

	done := make(chan result, 1)
	start := time.Now()
	go func() {
		got := result{probe: p, duration: time.Since(start)}
		data, err := linkparsers.ParseURL(p.url)
		got.duration = time.Since(start)
		if err != nil {
			got.status = statusError
			got.err = err.Error()
			done <- got
			return
		}
		got.title = strings.TrimSpace(data.Title)
		got.currency = data.Currency
		if data.Price != nil && *data.Price > 0 {
			got.price = fmt.Sprintf("%.2f", *data.Price)
		}
		switch {
		case got.title != "" && got.price != "":
			got.status = statusOK
		case got.title != "":
			got.status = statusNoPrice
		default:
			got.status = statusEmpty
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

func printTable(rows []result, verbose bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tHOST\tPARSER\tPRICE\tTIME")
	counts := map[status]int{}
	for _, r := range rows {
		counts[r.status]++
		price := r.price
		if price != "" && r.currency != "" {
			price = price + " " + r.currency
		}
		if price == "" {
			price = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.status, r.host, r.parser, price, r.duration.Truncate(time.Millisecond))
	}
	_ = w.Flush()

	fmt.Printf("\nworking %d  no_price %d  empty %d  error %d  timeout %d  no_probe %d  total %d\n",
		counts[statusOK], counts[statusNoPrice], counts[statusEmpty], counts[statusError], counts[statusTimeout], counts[statusNoProbe], len(rows))

	var needs []result
	for _, r := range rows {
		if r.status != statusOK {
			needs = append(needs, r)
		}
	}
	if len(needs) == 0 {
		fmt.Println("all probed parsers returned a title and price")
		return
	}
	fmt.Println("\nneeds attention:")
	for _, r := range needs {
		detail := r.err
		if detail == "" && r.status == statusNoProbe {
			detail = "no probe URL (a.co uses AmazonParser)"
		}
		if detail == "" && r.title != "" {
			detail = "title only: " + truncate(r.title, 80)
		}
		if detail == "" {
			detail = "no title or price"
		}
		fmt.Printf("  - %s (%s): %s %s\n", r.host, r.parser, r.status, truncate(detail, 140))
	}
	if verbose {
		fmt.Println("\ntitles:")
		for _, r := range rows {
			if r.title != "" {
				fmt.Printf("  %s  %s\n", r.host, truncate(r.title, 90))
			}
		}
	}
}

func unhealthy(rows []result) int {
	n := 0
	for _, r := range rows {
		if r.status != statusOK && r.status != statusNoProbe {
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
