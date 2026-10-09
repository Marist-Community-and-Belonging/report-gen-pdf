package reportgen

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// BuildFromProcessed turns the dashboard's processed extracts into report JSON
// without fetching IPEDS data again.
func BuildFromProcessed(processedDir string, year int) ([]byte, error) {
	return BuildFromFS(os.DirFS(processedDir), ".", year)
}

// BuildFromFS turns processed extracts in an fs.FS into report JSON. This lets
// applications generate reports directly from embedded dashboard data.
func BuildFromFS(content fs.FS, processedDir string, year int) ([]byte, error) {
	ov, err := loadProcessed(content, processedDir, "overview", year)
	if err != nil {
		return nil, err
	}
	metrics := asSlice(ov["metrics"])
	if len(metrics) == 0 {
		return nil, fmt.Errorf("overview-%d.json: no metrics", year)
	}
	div, err := loadProcessed(content, processedDir, "diversity-access", year)
	if err != nil {
		return nil, err
	}
	suc, err := loadProcessed(content, processedDir, "success-equity", year)
	if err != nil {
		return nil, err
	}
	aff, err := loadProcessed(content, processedDir, "affordability-resources", year)
	if err != nil {
		return nil, err
	}

	rel := asMap(ov["release"])
	collYear := str(rel["collection_year"])
	firstMetric := asMap(metrics[0])

	chapters := withFigures([]any{
		overviewChapter(ov),
		profileChapter(ov, div, suc, aff),
		trendChapter(content, processedDir, year),
		diversityChapter(div, content, processedDir, year),
		successChapter(suc),
		affordabilityChapter(aff),
	})

	doc := map[string]any{
		"title":    "Institutional Benchmarks",
		"subtitle": "Marist compared with peer and aspirant groups",
		"date":     time.Now().Format("January 2, 2006"),
		"author":   "Office of Community and Belonging",
		"office":   "Office of Community and Belonging",
		"phone":    "(845) 575-3000",
		"release": map[string]any{
			"collection_year": collYear,
			"release_type":    str(rel["release_type"]),
			"note":            "Retrieved " + str(rel["retrieved_at"]),
		},
		"intro": "Source data come from final NCES/IPEDS Complete Data Files for enrollment, admissions, retention, graduation, student aid, and net price. " +
			fmt.Sprintf(
				"The %s collection covers %.0f institutions: Marist, %.0f peers, and %.0f aspirants. Marist is excluded from comparison-group summaries. ",
				collYear,
				floatOf(ov["institution_count"]),
				floatOf(asMap(firstMetric["peer"])["count"]),
				floatOf(asMap(firstMetric["aspirant"])["count"]),
			) +
			"Each figure identifies its reporting period or cohort.",
		"chapters":       chapters,
		"references":     sourceReferences(ov),
		"reference_note": fmt.Sprintf("Final NCES/IPEDS Complete Data Files; retrieved %s. Comparison summaries exclude Marist and use institution-level values.", str(rel["retrieved_at"])),
		"source_url":     "https://nces.ed.gov/ipeds/use-the-data",
	}
	return json.MarshalIndent(doc, "", "  ")
}

// withFigures drops empty figures and the chapters left with none. The earliest
// collection has no prior years, so its trend charts have nothing to plot.
func withFigures(chapters []any) []any {
	kept := make([]any, 0, len(chapters))
	for _, raw := range chapters {
		c := asMap(raw)
		figs := make([]any, 0, len(asSlice(c["figures"])))
		for _, f := range asSlice(c["figures"]) {
			if len(asMap(f)) > 0 {
				figs = append(figs, f)
			}
		}
		if len(figs) == 0 {
			continue
		}
		c["figures"] = figs
		kept = append(kept, c)
	}
	return kept
}

func loadProcessed(content fs.FS, dir, name string, year int) (map[string]any, error) {
	file := path.Join(dir, fmt.Sprintf("%s-%d.json", name, year))
	b, err := fs.ReadFile(content, file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return m, nil
}

func overviewChapter(ov map[string]any) map[string]any {
	metrics := asSlice(ov["metrics"])
	figs := make([]any, 0, len(metrics))
	for _, raw := range metrics {
		m := asMap(raw)
		figs = append(figs, map[string]any{
			"kind":    "bullet",
			"title":   str(m["display_name"]),
			"unit":    mapUnit(str(m["unit"])),
			"caption": metricCaption(m),
			"value":   m["value"],
			"bands": []any{
				band("Peer", "peer", asMap(m["peer"])),
				band("Aspirant", "aspirant", asMap(m["aspirant"])),
			},
		})
	}
	return map[string]any{
		"heading": "Overview",
		"anchor":  "overview",
		"body":    "Higher is favorable except average net price, where lower is favorable.",
		"figures": collateBullets(figs),
		"callouts": []any{map[string]any{
			"label": "Reading IQR bullets",
			"body":  "Each shaded band spans Q1–Q3 for the comparison group. The solid tick is the median, the dashed tick is the mean, and the red line is Marist.",
		}},
	}
}

func trendChapter(content fs.FS, dir string, year int) map[string]any {
	// ponytail: fixed five-year window ending at selected year; widen if more collections land
	years := []int{}
	for y := year - 4; y <= year; y++ {
		years = append(years, y)
	}
	type series struct {
		marist, peer, asp []float64
		labels            []string
		unit              string
		title             string
	}
	byID := map[string]*series{}
	order := []string{}

	for _, y := range years {
		ov, err := loadProcessed(content, dir, "overview", y)
		if err != nil {
			continue
		}
		label := collectionLabel(asMap(ov["release"]))
		for _, raw := range asSlice(ov["metrics"]) {
			m := asMap(raw)
			id := str(m["metric_id"])
			if byID[id] == nil {
				byID[id] = &series{
					title: str(m["display_name"]),
					unit:  mapUnit(str(m["unit"])),
				}
				order = append(order, id)
			}
			s := byID[id]
			s.labels = append(s.labels, label)
			s.marist = append(s.marist, floatOf(m["value"]))
			s.peer = append(s.peer, floatOf(asMap(m["peer"])["median"]))
			s.asp = append(s.asp, floatOf(asMap(m["aspirant"])["median"]))
		}
	}

	figs := make([]any, 0, len(order))
	for _, id := range order {
		s := byID[id]
		if len(s.labels) < 2 {
			continue
		}
		figs = append(figs, map[string]any{
			"kind":     "line",
			"title":    s.title + ", multi-year",
			"unit":     s.unit,
			"x_labels": s.labels,
			"series": []any{
				map[string]any{"label": "Marist", "color": "marist", "values": s.marist},
				map[string]any{"label": "Peer median", "color": "peer", "values": s.peer},
				map[string]any{"label": "Aspirant median", "color": "aspirant", "values": s.asp},
			},
		})
	}
	return map[string]any{
		"heading": "Multi-year trends",
		"anchor":  "multi-year-trends",
		"body":    "Five final collections ending with the selected year.",
		"figures": figs,
	}
}

func diversityChapter(div map[string]any, content fs.FS, dir string, year int) map[string]any {
	cats := asSlice(div["categories"])
	catNames := make([]string, 0, len(cats))
	catIDs := make([]string, 0, len(cats))
	for _, c := range cats {
		cm := asMap(c)
		catIDs = append(catIDs, str(cm["category_id"]))
		catNames = append(catNames, str(cm["display_name"]))
	}

	series := []any{}
	colorFor := map[string]string{"marist": "marist", "peer": "peer", "aspirant": "aspirant"}
	for _, raw := range asSlice(div["groups"]) {
		g := asMap(raw)
		shares := asMap(g["normalized_shares"])
		reported := asMap(g["reported_shares"])
		vals := make([]any, 0, len(catIDs))
		displayVals := make([]any, 0, len(catIDs))
		for _, id := range catIDs {
			vals = append(vals, shares[id])
			displayVals = append(displayVals, reported[id])
		}
		grp := str(g["group"])
		series = append(series, map[string]any{
			"label":          str(g["display_name"]),
			"color":          colorFor[grp],
			"values":         vals,
			"display_values": displayVals,
		})
	}

	meth := asMap(div["methodology"])
	callouts := []any{}
	for _, raw := range asSlice(div["groups"]) {
		g := asMap(raw)
		if str(g["group"]) != "marist" {
			continue
		}
		shares := asMap(g["reported_shares"])
		underrepresented := floatOf(shares["black"]) + floatOf(shares["hispanic"]) +
			floatOf(shares["american_indian"]) + floatOf(shares["pacific_islander"]) +
			floatOf(shares["multiracial"])
		callouts = append(callouts, map[string]any{
			"label": "Underrepresented minority",
			"body": fmt.Sprintf(
				"Marist is %.1f%%. This combines Black, Hispanic or Latino, American Indian or Alaska Native, Native Hawaiian or Pacific Islander, and multiracial IPEDS categories.",
				underrepresented,
			),
		})
		break
	}
	for _, raw := range asSlice(div["institutions"]) {
		inst := asMap(raw)
		if str(inst["group"]) == "marist" {
			callouts = append(callouts, map[string]any{
				"label": "Diversity index",
				"body":  fmt.Sprintf("Marist is %.3f. %s", floatOf(inst["diversity_index"]), str(meth["diversity_index"])),
			})
			break
		}
	}
	return map[string]any{
		"heading": "Diversity and access",
		"anchor":  "diversity-access",
		"body":    "",
		"figures": []any{
			map[string]any{
				"kind":       "stacked",
				"title":      "Undergraduate composition",
				"unit":       "percent",
				"categories": catNames,
				"caption":    str(meth["composition"]),
				"series":     series,
			},
			accessFigure(div),
			diversityTrendFigure(content, dir, year),
			diversityScatter(div),
		},
		"callouts": callouts,
	}
}

// accessFigure mirrors the dashboard's three access-indicator cards.
func accessFigure(div map[string]any) map[string]any {
	groups := asSlice(div["groups"])
	indicators := []struct {
		label string
		pick  func(g map[string]any) any
	}{
		{"U.S. nonresident", func(g map[string]any) any { return asMap(g["reported_shares"])["nonresident"] }},
		{"Women", func(g map[string]any) any { return g["women_share"] }},
		{"Men", func(g map[string]any) any { return g["men_share"] }},
		{"Pell Grant recipients", func(g map[string]any) any { return g["pell_share"] }},
	}

	rows := []any{}
	for _, ind := range indicators {
		values := []any{}
		for _, raw := range groups {
			g := asMap(raw)
			values = append(values, map[string]any{
				"label": str(g["display_name"]),
				"value": ind.pick(g),
				"color": str(g["group"]),
			})
		}
		rows = append(rows, map[string]any{"label": ind.label, "values": values})
	}

	return map[string]any{
		"kind":    "grouped",
		"title":   "Access indicators",
		"unit":    "percent",
		"caption": "Group figures are enrollment-weighted aggregates, not medians.",
		"rows":    rows,
	}
}

// diversityTrendFigure tracks the diversity index, taking group medians across
// institutions exactly as the dashboard trend chart does.
func diversityTrendFigure(content fs.FS, dir string, year int) map[string]any {
	labels, marist, peer, aspirant := []any{}, []any{}, []any{}, []any{}

	for y := year - 4; y <= year; y++ {
		div, err := loadProcessed(content, dir, "diversity-access", y)
		if err != nil {
			continue
		}
		byGroup := map[string][]float64{}
		var own float64
		for _, raw := range asSlice(div["institutions"]) {
			inst := asMap(raw)
			if inst["diversity_index"] == nil {
				continue
			}
			g := str(inst["group"])
			if g == "marist" {
				own = floatOf(inst["diversity_index"])
				continue
			}
			byGroup[g] = append(byGroup[g], floatOf(inst["diversity_index"]))
		}
		labels = append(labels, collectionLabel(asMap(div["release"])))
		marist = append(marist, own)
		peer = append(peer, median(byGroup["peer"]))
		aspirant = append(aspirant, median(byGroup["aspirant"]))
	}
	if len(labels) < 2 {
		return nil
	}

	return map[string]any{
		"kind":     "line",
		"title":    "Diversity index, multi-year",
		"unit":     "index",
		"x_labels": labels,
		"caption":  "One minus the sum of squared category shares. Peer and aspirant are medians across institutions.",
		"series": []any{
			map[string]any{"label": "Marist", "color": "marist", "values": marist},
			map[string]any{"label": "Peer median", "color": "peer", "values": peer},
			map[string]any{"label": "Aspirant median", "color": "aspirant", "values": aspirant},
		},
	}
}

func diversityScatter(div map[string]any) map[string]any {
	points := []any{}
	for _, raw := range asSlice(div["institutions"]) {
		inst := asMap(raw)
		if inst["diversity_index"] == nil || inst["graduation_rate"] == nil || inst["enrollment"] == nil {
			continue
		}
		points = append(points, map[string]any{
			"label": str(inst["institution_name"]),
			"x":     inst["diversity_index"],
			"y":     inst["graduation_rate"],
			"size":  inst["enrollment"],
			"color": str(inst["group"]),
		})
	}

	return map[string]any{
		"kind":    "scatter",
		"title":   "Diversity and graduation",
		"x_label": "Diversity index",
		"y_label": "Six-year graduation rate",
		"x_unit":  "index",
		"y_unit":  "percent",
		"points":  points,
		"caption": "One mark per institution; area scales with undergraduate enrollment.",
		"legend": []any{
			map[string]any{"label": "Marist", "color": "marist"},
			map[string]any{"label": "Peer", "color": "peer"},
			map[string]any{"label": "Aspirant", "color": "aspirant"},
		},
	}
}

func successChapter(suc map[string]any) map[string]any {
	figs := []any{}
	for _, raw := range asSlice(suc["outcomes"]) {
		m := asMap(raw)
		if id := str(m["metric_id"]); id == "retention" || id == "six_year_graduation" {
			continue
		}
		figs = append(figs, map[string]any{
			"kind":    "bullet",
			"title":   str(m["display_name"]),
			"unit":    "percent",
			"caption": metricCaption(m),
			"value":   m["value"],
			"bands": []any{
				band("Peer", "peer", asMap(m["peer"])),
				band("Aspirant", "aspirant", asMap(m["aspirant"])),
			},
		})
	}
	figs = collateBullets(figs)

	gapRows := []any{}
	rateRows := []any{}
	section := ""
	for _, raw := range asSlice(suc["subgroups"]) {
		sg := asMap(raw)
		if sg["gap"] == nil {
			continue
		}
		if c := str(sg["category"]); c != section {
			section = c
			gapRows = append(gapRows, map[string]any{"label": c, "header": true, "values": []any{}})
			rateRows = append(rateRows, map[string]any{"label": c, "header": true, "values": []any{}})
		}
		gapRows = append(gapRows, map[string]any{
			"label": str(sg["display_name"]),
			"values": []any{
				map[string]any{"label": "", "value": sg["gap"], "color": "marist"},
			},
		})
		rateRows = append(rateRows, map[string]any{
			"label": str(sg["display_name"]),
			"values": []any{
				map[string]any{"label": "Marist", "value": sg["value"], "color": "marist"},
				map[string]any{"label": "Peer median", "value": asMap(sg["peer"])["median"], "mean": asMap(sg["peer"])["mean"], "color": "peer"},
				map[string]any{"label": "Aspirant median", "value": asMap(sg["aspirant"])["median"], "mean": asMap(sg["aspirant"])["mean"], "color": "aspirant"},
			},
		})
	}
	figs = append(figs, map[string]any{
		"kind":  "grouped",
		"title": "Equity gaps vs institution six-year rate",
		"unit":  "pp",
		"rows":  gapRows,
	})
	figs = append(figs, map[string]any{
		"kind":  "grouped",
		"title": "Subgroup six-year graduation rates",
		"unit":  "percent",
		"rows":  rateRows,
	})

	return map[string]any{
		"heading": "Success and equity",
		"anchor":  "success-equity",
		"body": fmt.Sprintf(
			"Additional outcomes for the %s context; retention and institution six-year graduation appear in Overview. Subgroup gaps are percentage points versus the institution six-year rate (%.0f%%).",
			str(suc["cohort_year"]), floatOf(suc["institution_rate"]),
		),
		"figures": figs,
	}
}

func collateBullets(figures []any) []any {
	grouped := make([]any, 0, (len(figures)+3)/4)
	for len(figures) > 0 {
		end := min(4, len(figures))
		grouped = append(grouped, map[string]any{"kind": "bullet-grid", "figures": figures[:end]})
		figures = figures[end:]
	}
	return grouped
}

func affordabilityChapter(aff map[string]any) map[string]any {
	figs := []any{}
	for _, raw := range asSlice(aff["headlines"]) {
		m := asMap(raw)
		if id := str(m["metric_id"]); id == "average_net_price" || id == "pell_share" {
			continue
		}
		figs = append(figs, map[string]any{
			"kind":    "bullet",
			"title":   str(m["display_name"]),
			"unit":    mapUnit(str(m["unit"])),
			"caption": metricCaption(m),
			"value":   m["value"],
			"bands": []any{
				band("Peer", "peer", asMap(m["peer"])),
				band("Aspirant", "aspirant", asMap(m["aspirant"])),
			},
		})
	}

	bandRows := []any{}
	for _, raw := range asSlice(aff["income_bands"]) {
		b := asMap(raw)
		bandRows = append(bandRows, map[string]any{
			"label": str(b["display_name"]),
			"values": []any{
				map[string]any{"label": "Marist", "value": b["marist"], "color": "marist"},
				map[string]any{"label": "Peer median", "value": asMap(b["peer"])["median"], "mean": asMap(b["peer"])["mean"], "color": "peer"},
				map[string]any{"label": "Aspirant median", "value": asMap(b["aspirant"])["median"], "mean": asMap(b["aspirant"])["mean"], "color": "aspirant"},
			},
		})
	}
	figs = append(figs, map[string]any{
		"kind":    "grouped",
		"title":   "Net price by family income band",
		"unit":    "usd",
		"caption": str(asMap(aff["methodology"])["income_bands"]),
		"rows":    bandRows,
	})

	return map[string]any{
		"heading": "Affordability",
		"anchor":  "affordability",
		"body":    "Average net price and Pell share appear in Overview.",
		"figures": figs,
	}
}

// profileChapter is the Marist-only view from the dashboard profile page.
// Its multi-year charts are left out because the trends chapter already plots
// the same overview metrics with peer and aspirant context.
func profileChapter(ov, div, suc, aff map[string]any) map[string]any {
	metrics := map[string]map[string]any{}
	for _, raw := range asSlice(ov["metrics"]) {
		m := asMap(raw)
		metrics[str(m["metric_id"])] = m
	}

	admissions := []any{}
	for _, id := range []string{"undergraduate_enrollment", "acceptance_rate"} {
		m := metrics[id]
		if m == nil {
			continue
		}
		admissions = append(admissions, statItem(
			str(m["display_name"]), m["value"], mapUnit(str(m["unit"])), str(m["data_year"]),
		))
	}

	outcomes := []any{}
	for _, raw := range asSlice(suc["outcomes"]) {
		m := asMap(raw)
		outcomes = append(outcomes, statItem(
			str(m["display_name"]), m["value"], "percent", str(m["data_year"]),
		))
	}

	costs := []any{}
	for _, raw := range asSlice(aff["headlines"]) {
		m := asMap(raw)
		costs = append(costs, statItem(
			str(m["display_name"]), m["value"], mapUnit(str(m["unit"])), str(m["data_year"]),
		))
	}
	if bands := asSlice(aff["income_bands"]); len(bands) > 0 {
		b := asMap(bands[0])
		costs = append(costs, statItem(
			"Net price, income "+str(b["display_name"]), b["marist"], "usd", str(aff["data_year"]),
		))
	}

	figs := []any{map[string]any{
		"kind":  "stats",
		"title": "Marist at a glance",
		"groups": []any{
			map[string]any{"title": "Enrollment and admissions", "items": admissions},
			map[string]any{"title": "Student outcomes", "items": outcomes},
			map[string]any{"title": "Affordability and access", "items": costs},
		},
	}}

	if fig := maristComposition(div); fig != nil {
		figs = append(figs, fig)
	}

	return map[string]any{
		"heading": "Marist profile",
		"anchor":  "marist-profile",
		"body":    "",
		"figures": figs,
	}
}

func maristComposition(div map[string]any) map[string]any {
	var marist map[string]any
	for _, raw := range asSlice(div["groups"]) {
		if g := asMap(raw); str(g["group"]) == "marist" {
			marist = g
			break
		}
	}
	if marist == nil {
		return nil
	}

	names, values := []string{}, []any{}
	shares := asMap(marist["normalized_shares"])
	for _, raw := range asSlice(div["categories"]) {
		c := asMap(raw)
		names = append(names, str(c["display_name"]))
		values = append(values, shares[str(c["category_id"])])
	}

	return map[string]any{
		"kind":       "stacked",
		"title":      "Marist undergraduate composition",
		"unit":       "percent",
		"categories": names,
		"caption":    str(asMap(div["methodology"])["composition"]),
		"series": []any{
			map[string]any{"label": "Marist", "color": "marist", "values": values},
		},
	}
}

func statItem(label string, value any, unit, note string) map[string]any {
	return map[string]any{"label": label, "value": statValue(value, unit), "note": note}
}

func metricCaption(m map[string]any) string {
	parts := []string{strings.TrimSuffix(str(m["description"]), ".")}
	if period := str(m["data_year"]); period != "" {
		parts = append(parts, "Period: "+period)
	}
	if cohort := str(m["cohort_year"]); cohort != "" {
		parts = append(parts, "Cohort: "+cohort)
	}
	if component, variable := str(m["source_component"]), str(m["source_variable"]); component != "" || variable != "" {
		parts = append(parts, "Source: "+strings.TrimSpace(component+" "+variable))
	}
	return strings.Join(parts, ". ") + "."
}

func sourceReferences(ov map[string]any) []any {
	metrics := map[string]map[string]any{}
	for _, raw := range asSlice(ov["metrics"]) {
		m := asMap(raw)
		metrics[str(m["metric_id"])] = m
	}
	source := func(id string) string {
		m := metrics[id]
		if m == nil {
			return ""
		}
		return strings.TrimSpace(str(m["source_component"]) + " " + str(m["source_variable"]))
	}
	return []any{
		map[string]any{
			"section":  "Overview",
			"anchor":   "overview",
			"source":   strings.Join([]string{source("undergraduate_enrollment"), source("acceptance_rate"), source("retention_rate"), source("six_year_graduation_rate"), source("pell_share"), source("average_net_price")}, "; "),
			"coverage": "Enrollment, admissions, retention, six-year graduation, Pell participation, and average net price.",
		},
		map[string]any{
			"section":  "Diversity and access",
			"anchor":   "diversity-access",
			"source":   source("undergraduate_enrollment") + " PCUENR*; " + source("six_year_graduation_rate") + "; " + source("pell_share"),
			"coverage": "Race/ethnicity composition, reported sex, access measures, diversity index, and graduation relationship.",
		},
		map[string]any{
			"section":  "Success and equity",
			"anchor":   "success-equity",
			"source":   source("retention_rate") + "; " + source("six_year_graduation_rate") + " GBA4RTT, GBA6RT*, PGBA6RT, SSBA6RT, NRBA6RT",
			"coverage": "Retention, four- and six-year completion, subgroup rates, and percentage-point gaps.",
		},
		map[string]any{
			"section":  "Affordability",
			"anchor":   "affordability",
			"source":   source("pell_share") + "; " + source("average_net_price") + " NPT412-NPT452",
			"coverage": "Pell participation, average net price, and net price by family-income band.",
		},
	}
}

func statValue(v any, unit string) string {
	f := floatOf(v)
	switch unit {
	case "usd":
		return "$" + commas(int64(f+0.5))
	case "percent":
		return fmt.Sprintf("%.1f%%", f)
	default:
		return commas(int64(f + 0.5))
	}
}

func commas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if n < 0 {
		return "-" + commas(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

func band(label, color string, g map[string]any) map[string]any {
	return map[string]any{
		"label":  label,
		"color":  color,
		"q1":     g["q1"],
		"q3":     g["q3"],
		"median": g["median"],
		"mean":   g["mean"],
	}
}

func mapUnit(u string) string {
	switch u {
	case "percent", "pct", "%":
		return "percent"
	case "usd", "dollars", "dollar":
		return "usd"
	case "students", "count":
		return "count"
	default:
		if u == "" {
			return "count"
		}
		return u
	}
}

func collectionLabel(rel map[string]any) string {
	if cy := str(rel["collection_year"]); cy != "" {
		return cy
	}
	return "?"
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func floatOf(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}
