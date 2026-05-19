// splitbandwith — Go re-implementation of splitbandwith.py
//
// Build:  go build -o splitbandwith ./cmd/splitbandwith
// Usage:  ./splitbandwith <source.csv> <domains.txt> [flags]
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/stanhui/splitbandwidth/internal/chart"
)

// ── helpers ──────────────────────────────────────────────────────────────────

var reUnsafe = regexp.MustCompile(`[^\w.\-]+`)

func safeFilename(name string) string {
	cleaned := reUnsafe.ReplaceAllString(strings.TrimSpace(name), "_")
	cleaned = strings.Trim(cleaned, "._")
	if cleaned == "" {
		return "domain"
	}
	return cleaned
}

func loadDomains(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := regexp.MustCompile(`[\s,;]+`).Split(strings.TrimSpace(string(raw)), -1)
	var domains []string
	for _, p := range parts {
		if p != "" {
			domains = append(domains, p)
		}
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("domain list is empty")
	}
	return domains, nil
}

func readTextFallback(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// strip UTF-8 BOM
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	return string(raw), nil
}

// columnIndex converts "A"→1, "B"→2, "AA"→27 …
func columnIndex(col string) (int, error) {
	val := 0
	for _, ch := range strings.ToUpper(col) {
		if ch < 'A' || ch > 'Z' {
			return 0, fmt.Errorf("invalid column name: %s", col)
		}
		val = val*26 + int(ch-'A') + 1
	}
	return val, nil
}

func zeroBasedColumnIndex(col string) (int, error) {
	allDigit := true
	for _, ch := range col {
		if !unicode.IsDigit(ch) {
			allDigit = false
			break
		}
	}
	if allDigit {
		n, err := strconv.Atoi(col)
		if err != nil || n < 1 {
			return 0, fmt.Errorf("column number must be >= 1")
		}
		return n - 1, nil
	}
	idx, err := columnIndex(col)
	if err != nil {
		return 0, err
	}
	return idx - 1, nil
}

func resolveBandwidthIndex(header []string, bwCol string) (int, error) {
	for i, h := range header {
		if h == bwCol {
			return i, nil
		}
	}
	return zeroBasedColumnIndex(bwCol)
}

// ── CSV reading ───────────────────────────────────────────────────────────────

type sourceData struct {
	header     []string
	rows       [][]string
	bwIndex    int
	values     []float64
	useSemicolon bool
}

func readCSVSource(path, bwCol string) (*sourceData, error) {
	text, err := readTextFallback(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("source csv is empty")
	}

	// detect delimiter: count semicolons vs commas in first few lines
	sample := strings.Join(lines[:min(20, len(lines))], "\n")
	useSemicolon := strings.Count(sample, ";") > strings.Count(sample, ",")

	delim := ','
	if useSemicolon {
		delim = ';'
	}

	r := csv.NewReader(strings.NewReader(text))
	r.Comma = rune(delim)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	all, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(all) < 2 {
		return nil, fmt.Errorf("source csv must contain a header row and data rows")
	}

	header := all[0]
	dataRows := all[1:]

	bwIdx, err := resolveBandwidthIndex(header, bwCol)
	if err != nil {
		return nil, err
	}
	if bwIdx >= len(header) {
		return nil, fmt.Errorf("bandwidth column out of range: %s", bwCol)
	}

	values := make([]float64, len(dataRows))
	for i, row := range dataRows {
		if bwIdx >= len(row) || strings.TrimSpace(row[bwIdx]) == "" {
			return nil, fmt.Errorf("missing bandwidth value at csv row %d", i+2)
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(row[bwIdx]), 64)
		if err != nil {
			return nil, fmt.Errorf("csv row %d bandwidth is not a number: %s", i+2, row[bwIdx])
		}
		values[i] = v
	}

	return &sourceData{
		header:       header,
		rows:         dataRows,
		bwIndex:      bwIdx,
		values:       values,
		useSemicolon: useSemicolon,
	}, nil
}

// ── splitting ─────────────────────────────────────────────────────────────────

func splitOneValue(total float64, weights []float64, rng *rand.Rand, scale float64) []float64 {
	n := len(weights)
	if n == 1 {
		return []float64{total}
	}
	weightSum := 0.0
	for _, w := range weights {
		weightSum += w
	}

	totalUnits := math.Floor(total / scale)
	remainder := total - totalUnits*scale
	rawUnits := make([]float64, n)
	units := make([]int, n)
	unitSum := 0
	for i, w := range weights {
		rawUnits[i] = totalUnits * w / weightSum
		units[i] = int(math.Floor(rawUnits[i]))
		unitSum += units[i]
	}
	leftover := int(totalUnits) - unitSum

	// distribute leftover to indices with largest fractional parts
	type idxFrac struct {
		idx  int
		frac float64
	}
	fracs := make([]idxFrac, n)
	for i := range units {
		fracs[i] = idxFrac{i, rawUnits[i] - float64(units[i])}
	}
	// simple selection for leftover (usually small)
	for k := 0; k < leftover; k++ {
		best := 0
		for j := 1; j < n; j++ {
			if fracs[j].frac > fracs[best].frac {
				best = j
			}
		}
		units[fracs[best].idx]++
		fracs[best].frac = -1
	}

	parts := make([]float64, n)
	for i, u := range units {
		parts[i] = float64(u) * scale
	}
	parts[rng.Intn(n)] += remainder
	return parts
}

func randomWeights(n int, rng *rand.Rand) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = float64(rng.Intn(1_000_000) + 1)
	}
	return w
}

// profileWeightRows generates correlated weight rows using an AR(1) log-normal process.
func profileWeightRows(rowCount, domainCount int, rng *rand.Rand, domainSpread, volatility, smoothness float64) [][]float64 {
	baseWeights := make([]float64, domainCount)
	logStates := make([]float64, domainCount)
	for i := range baseWeights {
		baseWeights[i] = math.Exp(rng.NormFloat64() * domainSpread)
		logStates[i] = rng.NormFloat64() * volatility
	}
	rho := math.Min(math.Max(smoothness, 0), 0.999)
	shockScale := volatility * math.Sqrt(math.Max(1-rho*rho, 1e-6))

	result := make([][]float64, rowCount)
	for r := range result {
		row := make([]float64, domainCount)
		for i := range row {
			logStates[i] = rho*logStates[i] + rng.NormFloat64()*shockScale
			row[i] = baseWeights[i] * math.Exp(logStates[i])
		}
		result[r] = row
	}
	return result
}

func splitValues(values []float64, domainCount int, seed int64, hasSeed bool, scale float64, mode string, domainSpread, volatility, smoothness float64) ([][]float64, error) {
	var src rand.Source
	if hasSeed {
		src = rand.NewSource(seed)
	} else {
		src = rand.NewSource(rand.Int63())
	}
	rng := rand.New(src)

	perDomain := make([][]float64, domainCount)
	for i := range perDomain {
		perDomain[i] = make([]float64, len(values))
	}

	var weightRows [][]float64
	switch mode {
	case "independent":
		weightRows = make([][]float64, len(values))
		for i := range weightRows {
			weightRows[i] = randomWeights(domainCount, rng)
		}
	case "profile":
		weightRows = profileWeightRows(len(values), domainCount, rng, domainSpread, volatility, smoothness)
	default:
		return nil, fmt.Errorf("unknown split mode: %s", mode)
	}

	for i, total := range values {
		if total < 0 {
			return nil, fmt.Errorf("bandwidth value must be >= 0: %v", total)
		}
		parts := splitOneValue(total, weightRows[i], rng, scale)
		for d, p := range parts {
			perDomain[d][i] = p
		}
	}
	return perDomain, nil
}

// ── CSV writing ───────────────────────────────────────────────────────────────

func formatValue(v float64, decimalPlaces int) string {
	if decimalPlaces == 0 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', decimalPlaces, 64)
}

func writeCSV(path string, header []string, rows [][]string, useSemicolon bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	// write UTF-8 BOM for Excel compatibility
	f.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(f)
	if useSemicolon {
		w.Comma = ';'
	}
	w.Write(header)
	w.WriteAll(rows)
	return w.Error()
}

func outputCSVMany(sd *sourceData, domains []string, perDomain [][]float64, outputDir string, decimalPlaces int) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	used := map[string]bool{}
	for di, domain := range domains {
		base := safeFilename(domain)
		filename := base + ".csv"
		for suffix := 2; used[filename]; suffix++ {
			filename = fmt.Sprintf("%s_%d.csv", base, suffix)
		}
		used[filename] = true

		newRows := make([][]string, len(sd.rows))
		for ri, row := range sd.rows {
			nr := make([]string, len(sd.header))
			copy(nr, row)
			for len(nr) < len(sd.header) {
				nr = append(nr, "")
			}
			nr[sd.bwIndex] = formatValue(perDomain[di][ri], decimalPlaces)
			newRows[ri] = nr
		}
		if err := writeCSV(filepath.Join(outputDir, filename), sd.header, newRows, sd.useSemicolon); err != nil {
			return err
		}
	}
	return nil
}

func outputCSVSingle(sd *sourceData, domains []string, perDomain [][]float64, outputFile string, decimalPlaces int) error {
	header := append(append([]string{}, sd.header...), domains...)
	rows := make([][]string, len(sd.rows))
	for ri, row := range sd.rows {
		nr := make([]string, len(sd.header))
		copy(nr, row)
		for len(nr) < len(sd.header) {
			nr = append(nr, "")
		}
		for di := range domains {
			nr = append(nr, formatValue(perDomain[di][ri], decimalPlaces))
		}
		rows[ri] = nr
	}
	return writeCSV(outputFile, header, rows, sd.useSemicolon)
}

// ── chart ─────────────────────────────────────────────────────────────────────

func writeChartHTML(path string, sd *sourceData, domains []string, perDomain [][]float64) error {
	timeIdx := 0
	if sd.bwIndex == 0 {
		timeIdx = 1
	}
	if timeIdx >= len(sd.header) {
		return fmt.Errorf("cannot find a time column for chart")
	}

	xVals := make([]string, len(sd.rows))
	for i, row := range sd.rows {
		if timeIdx < len(row) {
			xVals[i] = row[timeIdx]
		}
	}

	series := make([]chart.Series, len(domains))
	for i, d := range domains {
		pk := 0.0
		for _, v := range perDomain[i] {
			if v > pk {
				pk = v
			}
		}
		series[i] = chart.Series{Name: d, Values: perDomain[i], Peak: pk}
	}

	return chart.WriteHTML(path, chart.Config{
		Title:   "Bandwidth Split by Domain",
		XLabels: xVals,
		Total:   sd.values,
		Series:  series,
	})
}

// ── main ──────────────────────────────────────────────────────────────────────

type boolFlag interface{ IsBoolFlag() bool }

func main() {
	fs := flag.NewFlagSet("splitbandwith", flag.ExitOnError)
	outputDir := fs.String("output-dir", "split_output", "output directory (one CSV per domain)")
	outputFile := fs.String("o", "", "single output CSV (all domains as columns)")
	outputFileLong := fs.String("output-file", "", "same as -o")
	chartFile := fs.String("chart-file", "", "Plotly HTML chart path")
	noChart := fs.Bool("no-chart", false, "skip generating HTML chart")
	bwCol := fs.String("bandwidth-col", "B", "bandwidth column: header name, number, or letter")
	seedVal := fs.Int64("seed", 0, "random seed (0 = random)")
	hasSeed := false
	decimalPlaces := fs.Int("decimal-places", 0, "decimal places for output values")
	mode := fs.String("mode", "profile", "split mode: profile or independent")
	domainSpread := fs.Float64("domain-spread", 1.2, "profile: spread between domains")
	volatility := fs.Float64("volatility", 0.18, "profile: time-varying volatility")
	smoothness := fs.Float64("smoothness", 0.98, "profile: smoothness (< 1)")

	// Reorder argv so flags come before positional args, allowing mixed order:
	//   splitbandwith source.csv domains.txt -o out.csv
	var flagArgs, posArgs []string
	rawArgs := os.Args[1:]
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		if strings.HasPrefix(a, "-") {
			if a == "--seed" || a == "-seed" ||
				strings.HasPrefix(a, "--seed=") || strings.HasPrefix(a, "-seed=") {
				hasSeed = true
			}
			flagArgs = append(flagArgs, a)
			// if flag has no '=', consume next token as its value (for non-bool flags)
			if !strings.Contains(a, "=") && i+1 < len(rawArgs) {
				name := strings.TrimLeft(a, "-")
				if f := fs.Lookup(name); f != nil {
					if _, isBool := f.Value.(boolFlag); !isBool {
						i++
						flagArgs = append(flagArgs, rawArgs[i])
					}
				}
			}
		} else {
			posArgs = append(posArgs, a)
		}
	}
	fs.Parse(append(flagArgs, posArgs...))
	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: splitbandwith <source.csv> <domains.txt> [flags]")
		fs.PrintDefaults()
		os.Exit(1)
	}

	outFile := *outputFile
	if outFile == "" {
		outFile = *outputFileLong
	}

	if *decimalPlaces < 0 {
		fmt.Fprintln(os.Stderr, "--decimal-places must be >= 0")
		os.Exit(1)
	}
	if *domainSpread < 0 || *volatility < 0 || *smoothness < 0 || *smoothness >= 1 {
		fmt.Fprintln(os.Stderr, "invalid profile parameters")
		os.Exit(1)
	}

	sourcePath := args[0]
	domainsPath := args[1]

	if strings.ToLower(filepath.Ext(sourcePath)) != ".csv" {
		fmt.Fprintln(os.Stderr, "source file must be .csv")
		os.Exit(1)
	}

	domains, err := loadDomains(domainsPath)
	check(err)

	sd, err := readCSVSource(sourcePath, *bwCol)
	check(err)

	scale := math.Pow(10, -float64(*decimalPlaces))
	perDomain, err := splitValues(sd.values, len(domains), *seedVal, hasSeed, scale, *mode, *domainSpread, *volatility, *smoothness)
	check(err)

	if outFile != "" {
		check(outputCSVSingle(sd, domains, perDomain, outFile, *decimalPlaces))
		if !*noChart {
			cf := *chartFile
			if cf == "" {
				cf = strings.TrimSuffix(outFile, filepath.Ext(outFile)) + ".html"
			}
			check(writeChartHTML(cf, sd, domains, perDomain))
			fmt.Println("chart file:", cf)
		}
		fmt.Println("output file:", outFile)
	} else {
		check(outputCSVMany(sd, domains, perDomain, *outputDir, *decimalPlaces))
		fmt.Println("output dir:", *outputDir)
	}

	fmt.Printf("source rows: %d\n", len(sd.values))
	fmt.Printf("domains: %d\n", len(domains))
	shown := domains
	if len(shown) > 20 {
		shown = shown[:20]
	}
	for _, d := range shown {
		fmt.Println("-", d)
	}
	if len(domains) > len(shown) {
		fmt.Printf("... %d more domains\n", len(domains)-len(shown))
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
