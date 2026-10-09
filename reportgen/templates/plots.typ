// Shared CeTZ helpers for dashboard-style comparison charts.
// Canvas units are centimetres; PW matches the 6in text column.
#import "@preview/cetz:0.3.4"
#import cetz.draw: line, rect, circle, content

#let PW = 15.2

#let brand = (
  marist: rgb("#c91235"),
  peer: rgb("#63666f"),
  aspirant: rgb("#3c8cff"),
  ink: rgb("#202127"),
  muted: rgb("#63666f"),
  grid: rgb("#eceef1"),
  axis: rgb("#d8dade"),
)

#let category-palette = (
  rgb("#8e2038"),
  rgb("#3e5968"),
  rgb("#b35d38"),
  rgb("#596b3d"),
  rgb("#705780"),
  rgb("#9a7a28"),
  rgb("#3f7771"),
  rgb("#777982"),
  rgb("#3c8cff"),
)

#let series-fill(name) = brand.at(name, default: brand.ink)

#let thousands(v) = {
  let s = str(calc.round(calc.abs(v)))
  let digits = s.clusters()
  let out = ""
  for (i, ch) in digits.enumerate() {
    out += ch
    let rem = digits.len() - i - 1
    if rem > 0 and calc.rem(rem, 3) == 0 { out += "," }
  }
  if v < 0 { "-" + out } else { out }
}

// Full-precision label for a data point.
#let fmt-val(v, unit) = {
  let f = float(v)
  if unit == "usd" {
    let s = thousands(f)
    if f < 0 { "-$" + s.slice(1) } else { "$" + s }
  } else if unit == "percent" {
    str(calc.round(f, digits: 1)) + "%"
  } else if unit == "pp" {
    (if f > 0 { "+" } else { "" }) + str(calc.round(f, digits: 1)) + " pp"
  } else if unit == "index" {
    str(calc.round(f, digits: 3))
  } else if calc.abs(f) >= 1000 {
    thousands(f)
  } else {
    str(calc.round(f, digits: 1))
  }
}

// Compact label for an axis tick.
#let tick-label(v, unit) = {
  let f = float(v)
  if unit == "usd" {
    if calc.abs(f) >= 1000 { "$" + str(calc.round(f / 1000)) + "k" } else { "$" + str(calc.round(f)) }
  } else if unit == "percent" {
    str(calc.round(f, digits: 1)) + "%"
  } else if unit == "index" {
    str(calc.round(f, digits: 2))
  } else if calc.abs(f) >= 10000 {
    str(calc.round(f / 1000)) + "k"
  } else if calc.abs(f) >= 1000 {
    str(calc.round(f / 1000, digits: 1)) + "k"
  } else {
    str(calc.round(f, digits: 1))
  }
}

// Headroom must not invent impossible values (a share above 100%, a negative count).
#let clamp-bounds(lo, hi, unit, data-lo, data-hi) = {
  let l = if data-lo >= 0 { calc.max(lo, 0.0) } else { lo }
  let h = if unit == "percent" and data-hi <= 100 { calc.min(hi, 100.0) } else { hi }
  (l, h)
}

#let nice-step(span, target) = {
  if span <= 0 { return 1.0 }
  let raw = span / target
  let mag = calc.pow(10.0, calc.floor(calc.log(raw, base: 10)))
  let norm = raw / mag
  let mult = if norm <= 1.0 { 1.0 } else if norm <= 2.0 { 2.0 } else if norm <= 5.0 { 5.0 } else { 10.0 }
  mult * mag
}

// Rounded axis bounds + tick positions covering [vmin, vmax].
// Tries progressively finer steps so rounding never leaves the plot mostly empty.
#let axis-scale(vmin, vmax, target: 5) = {
  let want = calc.max(vmax - vmin, 0.000001)
  let step = nice-step(want, target)
  for t in (target, target + 2, target + 4) {
    let s = nice-step(want, t)
    step = s
    let lo = calc.floor(vmin / s) * s
    let hi = calc.ceil(vmax / s) * s
    if hi - lo <= want * 1.8 and (hi - lo) / s <= 8 { break }
  }
  let lo = calc.floor(vmin / step) * step
  let hi = calc.ceil(vmax / step) * step
  if hi - lo < step / 2 { hi = lo + step }
  let ticks = ()
  let t = lo
  let guard = 0
  while t <= hi + step / 1000 and guard < 40 {
    ticks.push(t)
    t += step
    guard += 1
  }
  (lo: lo, hi: hi, span: calc.max(hi - lo, 0.000001), ticks: ticks)
}

// Vertical gridlines with labels under the plot (horizontal charts).
#let x-grid(x0, y0, w, h, sc, unit) = {
  for t in sc.ticks {
    let x = x0 + ((t - sc.lo) / sc.span) * w
    line((x, y0), (x, y0 + h), stroke: 0.4pt + brand.grid)
    content(
      (x, y0 - 0.16),
      text(font: "Inter", size: 6.5pt, fill: brand.muted)[#tick-label(t, unit)],
      anchor: "north",
    )
  }
  line((x0, y0), (x0 + w, y0), stroke: 0.5pt + brand.axis)
}

// Horizontal gridlines with labels left of the plot (vertical charts).
#let y-grid(x0, y0, w, h, sc, unit) = {
  for t in sc.ticks {
    let y = y0 + ((t - sc.lo) / sc.span) * h
    line((x0, y), (x0 + w, y), stroke: 0.4pt + brand.grid)
    content(
      (x0 - 0.16, y),
      text(font: "Inter", size: 6.5pt, fill: brand.muted)[#tick-label(t, unit)],
      anchor: "east",
    )
  }
  line((x0, y0), (x0, y0 + h), stroke: 0.5pt + brand.axis)
}

#let fig-title(figure, extra: none) = {
  if figure.at("title", default: "") != "" {
    text(font: "Inter", size: 10pt, weight: "semibold", fill: brand.ink)[#figure.title]
    if extra != none [#h(8pt)#extra]
    v(8pt)
  }
}

#let fig-caption(figure) = {
  if figure.at("caption", default: "") != "" {
    v(5pt)
    text(font: "Inter", size: 8.5pt, fill: brand.muted)[#figure.caption]
  }
}

#let swatch-legend(items) = {
  v(6pt)
  set par(leading: 0.75em)
  // Outer box keeps a swatch glued to its label across line wraps.
  for it in items [
    #box[
      #box(baseline: 1pt, width: 7pt, height: 7pt, radius: 1pt, fill: it.color)
      #h(3.5pt)
      #text(font: "Inter", size: 7.5pt, fill: brand.muted)[#it.label]
    ]
    #h(11pt)
  ]
}

// Wrapped right-aligned row label in the left gutter.
#let row-label(x, y, w, body, size: 6.5pt) = {
  content(
    (x, y),
    box(width: w * 1cm, inset: (right: 4pt))[
      #set par(leading: 0.45em, justify: false)
      #align(right, text(font: "Inter", size: size, fill: brand.muted)[#body])
    ],
    anchor: "east",
  )
}

// —— Bullet: Marist mark against peer/aspirant IQR bands ——
#let bullet-iqr(figure, compact: false) = {
  let bands = figure.at("bands", default: ()).filter(b => b.at("q1", default: none) != none)
  if bands.len() == 0 { return }
  let unit = figure.at("unit", default: "")
  let val = float(figure.at("value", default: 0))

  let nums = (val,)
  for b in bands {
    nums.push(float(b.q1))
    nums.push(float(b.q3))
    nums.push(float(b.median))
    if b.at("mean", default: none) != none { nums.push(float(b.mean)) }
  }
  let lo = calc.min(..nums)
  let hi = calc.max(..nums)
  // Padding stays proportional; an absolute floor would swamp a 0-1 index.
  let pad = calc.max((hi - lo) * 0.18, calc.abs(hi) * 0.02, 0.000001)
  let (blo, bhi) = clamp-bounds(lo - pad, hi + pad, unit, lo, hi)
  let sc = axis-scale(blo, bhi, target: 5)

  let total-w = if compact { 7.25 } else { PW }
  let gutter-l = if compact { 1.25 } else { 2.0 }
  let gutter-r = if compact { 1.55 } else { 2.6 }
  let plot-w = total-w - gutter-l - gutter-r
  let row-h = 0.62
  let chart-h = bands.len() * row-h

  let px(v) = gutter-l + ((float(v) - sc.lo) / sc.span) * plot-w

  block(width: 100%, breakable: false)[
    #fig-title(
      figure,
      extra: text(font: "Inter", size: 10pt, weight: "bold", fill: brand.marist)[
        Marist #fmt-val(val, unit)
      ],
    )
    #cetz.canvas({
      x-grid(gutter-l, 0, plot-w, chart-h, sc, unit)

      for (i, b) in bands.enumerate() {
        let y = chart-h - (i + 0.5) * row-h
        let c = series-fill(b.at("color", default: "peer"))
        rect((px(b.q1), y - 0.17), (px(b.q3), y + 0.17), fill: c.lighten(62%), stroke: none)
        line((px(b.median), y - 0.21), (px(b.median), y + 0.21), stroke: 1.8pt + c)
        if b.at("mean", default: none) != none {
          line(
            (px(b.mean), y - 0.17),
            (px(b.mean), y + 0.17),
            stroke: (paint: c, thickness: 1.2pt, dash: "dashed"),
          )
        }
        row-label(gutter-l, y, gutter-l, b.at("label", default: ""), size: if compact { 6.5pt } else { 7.5pt })
        content(
          (gutter-l + plot-w + 0.18, y),
          stack(
            dir: ttb,
            spacing: 1.2pt,
            text(font: "Inter", size: if compact { 5.7pt } else { 6.5pt }, fill: brand.muted)[
              med #fmt-val(b.median, unit)
            ],
            if b.at("mean", default: none) == none { none } else {
              text(font: "Inter", size: if compact { 5.7pt } else { 6.5pt }, fill: brand.muted)[
                avg #fmt-val(b.mean, unit)
              ]
            },
          ),
          anchor: "west",
        )
      }

      // Marist reference line across every band.
      line((px(val), 0), (px(val), chart-h + 0.16), stroke: 1.6pt + brand.marist)
      content(
        (px(val), chart-h + 0.22),
        text(font: "Inter", size: 6.5pt, weight: "bold", fill: brand.marist)[Marist],
        anchor: "south",
      )
    })
    #fig-caption(figure)
  ]
}

#let bullet-grid(figure) = {
  let figures = figure.at("figures", default: ())
  if figures.len() == 0 { return }
  grid(
    columns: (1fr, 1fr),
    gutter: 14pt,
    row-gutter: 12pt,
    ..figures.map(figure => bullet-iqr(figure, compact: true)),
  )
}

// —— Line: multi-year trend with direct series labels ——
#let line-trend(figure) = {
  let series = figure.at("series", default: ())
  let labels = figure.at("x_labels", default: ())
  if series.len() == 0 or labels.len() == 0 { return }
  let unit = figure.at("unit", default: "")

  let all = series.map(s => s.values.map(float)).flatten()
  let lo = calc.min(..all)
  let hi = calc.max(..all)
  let pad = calc.max((hi - lo) * 0.15, calc.abs(hi) * 0.02, 0.000001)
  let (blo, bhi) = clamp-bounds(lo - pad, hi + pad, unit, lo, hi)
  let sc = axis-scale(blo, bhi, target: 4)

  let gutter-l = 1.4
  let gutter-r = 3.0
  let plot-w = PW - gutter-l - gutter-r
  let chart-h = 3.4
  let n = labels.len()
  let step = if n > 1 { plot-w / (n - 1) } else { plot-w }

  let py(v) = ((float(v) - sc.lo) / sc.span) * chart-h

  // Direct labels at the final point, nudged apart on collision.
  let ends = series
    .map(s => (
      y: py(s.values.last()),
      label: s.at("label", default: ""),
      color: series-fill(s.at("color", default: "ink")),
    ))
    .sorted(key: e => e.y)
  let placed = ()
  let prev = -99.0
  for e in ends {
    let y = calc.max(e.y, prev + 0.38)
    placed.push((..e, y: y))
    prev = y
  }

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #cetz.canvas({
      y-grid(gutter-l, 0, plot-w, chart-h, sc, unit)
      line((gutter-l, 0), (gutter-l + plot-w, 0), stroke: 0.5pt + brand.axis)

      for s in series {
        let c = series-fill(s.at("color", default: "ink"))
        let pts = s.values.enumerate().map(((i, v)) => (gutter-l + i * step, py(v)))
        for i in range(pts.len() - 1) {
          line(pts.at(i), pts.at(i + 1), stroke: 1.6pt + c)
        }
        for p in pts {
          circle(p, radius: 0.07, fill: c, stroke: 1pt + white)
        }
      }

      // End labels hug the axis so they never spill past the plot.
      for (i, lab) in labels.enumerate() {
        content(
          (gutter-l + i * step, -0.16),
          text(font: "Inter", size: 7pt, fill: brand.muted)[#lab],
          anchor: if i == 0 { "north-west" } else if i == n - 1 { "north-east" } else { "north" },
        )
      }

      for p in placed {
        content(
          (gutter-l + plot-w + 0.22, p.y),
          text(font: "Inter", size: 7pt, weight: "semibold", fill: p.color)[#p.label],
          anchor: "west",
        )
      }
    })
    #fig-caption(figure)
  ]
}

// —— Horizontal grouped bars: one row per category, N bars per row ——
#let grouped-bars(figure) = {
  let rows = figure.at("rows", default: ())
  if rows.len() == 0 { return }
  let unit = figure.at("unit", default: "")

  let nums = rows.map(r => r.values.map(v => (
    float(v.value),
    ..if v.at("mean", default: none) == none { () } else { (float(v.mean),) },
  )).flatten()).flatten()
  if nums.len() == 0 { return }
  let nv = calc.max(..rows.map(r => r.values.len()))
  let lo = calc.min(..nums, 0.0)
  let hi = calc.max(..nums, 0.0)
  let headroom = calc.max((hi - lo) * 0.14, 0.001)
  let (blo, bhi) = clamp-bounds(
    if lo < 0 { lo - headroom } else { lo },
    hi + headroom,
    unit,
    lo,
    hi,
  )
  let sc = axis-scale(blo, bhi, target: 5)

  let bar-h = if nv > 1 { 0.26 } else { 0.30 }
  let row-h = calc.max(nv * bar-h + 0.30, 0.62)
  let chart-h = rows.len() * row-h
  let gutter-l = 3.9
  let gutter-r = if sc.lo >= 0 { 1.6 } else { 0.3 }
  let plot-w = PW - gutter-l - gutter-r

  let px(v) = gutter-l + ((float(v) - sc.lo) / sc.span) * plot-w
  let zero = px(calc.max(calc.min(0.0, sc.hi), sc.lo))

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #cetz.canvas({
      x-grid(gutter-l, 0, plot-w, chart-h, sc, unit)
      if sc.lo < 0 {
        line((zero, 0), (zero, chart-h), stroke: 0.7pt + brand.axis)
      }

      for (ri, r) in rows.enumerate() {
        // Centre each bar group on its row so the gutter label lines up.
        let mid = chart-h - (ri + 0.5) * row-h
        let first = mid + (r.values.len() - 1) / 2 * bar-h

        if r.at("header", default: false) {
          content(
            (gutter-l - 0.1, mid),
            text(font: "Inter", size: 7pt, weight: "bold", fill: brand.ink)[
              #r.at("label", default: "")
            ],
            anchor: "east",
          )
          line(
            (gutter-l, mid + row-h / 2),
            (gutter-l + plot-w, mid + row-h / 2),
            stroke: 0.5pt + brand.axis,
          )
          continue
        }
        row-label(gutter-l, mid, gutter-l, r.at("label", default: ""))

        for (vi, v) in r.values.enumerate() {
          let y = first - vi * bar-h
          let x1 = px(v.value)
          let c = series-fill(v.at("color", default: "marist"))
          rect((zero, y - bar-h * 0.38), (x1, y + bar-h * 0.38), fill: c, stroke: none)
          if v.at("mean", default: none) != none {
            circle(
              (px(v.mean), y),
              radius: 0.06,
              fill: white,
              stroke: 1pt + c.darken(15%),
            )
          }
          let label-x = if sc.lo >= 0 {
            gutter-l + plot-w + gutter-r
          } else {
            x1 + (if float(v.value) < 0 { -0.22 } else { 0.22 })
          }
          content(
            (label-x, y),
            box(
              inset: (left: 1pt, right: 1pt),
              fill: white,
              text(font: "Inter", size: 6.2pt, weight: "semibold", fill: brand.ink)[
                #fmt-val(v.value, unit)
              ],
            ),
            anchor: if sc.lo >= 0 or float(v.value) < 0 { "east" } else { "west" },
          )
        }
      }
    })
    #if nv > 1 {
      swatch-legend(rows.find(r => r.values.len() == nv).values.map(v => (
        label: v.at("label", default: ""),
        color: series-fill(v.at("color", default: "marist")),
      )))
    }
    #if rows.any(r => r.values.any(v => v.at("mean", default: none) != none)) [
      #text(font: "Inter", size: 7pt, fill: brand.muted)[Hollow dots show comparison-group means.]
    ]
    #fig-caption(figure)
  ]
}

// —— Vertical bars: short comparisons (hand-authored reports) ——
#let bar-compare(figure) = {
  let series = figure.at("series", default: ()).filter(s => s.at("value", default: none) != none)
  if series.len() == 0 { return }
  let unit = figure.at("unit", default: "")
  let n = series.len()
  let longest = calc.max(..series.map(s => str(s.at("label", default: "")).len()), 1)

  // Long or numerous labels never fit under vertical bars.
  if n > 6 or longest > 16 {
    return grouped-bars((
      ..figure,
      rows: series.map(s => (
        label: s.at("label", default: ""),
        values: ((label: "", value: s.value, color: s.at("color", default: "marist")),),
      )),
    ))
  }

  let vals = series.map(s => float(s.value))
  let lo = calc.min(..vals, 0.0)
  let hi = calc.max(..vals, 0.0)
  let (blo, bhi) = clamp-bounds(lo, hi + calc.max((hi - lo) * 0.12, 0.001), unit, lo, hi)
  let sc = axis-scale(blo, bhi, target: 4)

  let gutter-l = 1.4
  let plot-w = PW - gutter-l - 0.3
  let chart-h = 3.4
  let slot = plot-w / n
  let bar-w = calc.min(1.7, slot * 0.5)

  let py(v) = ((float(v) - sc.lo) / sc.span) * chart-h
  let zero = py(calc.max(calc.min(0.0, sc.hi), sc.lo))

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #cetz.canvas({
      y-grid(gutter-l, 0, plot-w, chart-h, sc, unit)
      line((gutter-l, zero), (gutter-l + plot-w, zero), stroke: 0.5pt + brand.axis)

      for (i, s) in series.enumerate() {
        let cx = gutter-l + (i + 0.5) * slot
        let h = py(s.value)
        let c = series-fill(s.at("color", default: "marist"))
        rect((cx - bar-w / 2, zero), (cx + bar-w / 2, h), fill: c, stroke: none)
        content(
          (cx, h + (if h >= zero { 0.12 } else { -0.12 })),
          text(font: "Inter", size: 7.5pt, weight: "semibold", fill: brand.ink)[
            #fmt-val(s.value, unit)
          ],
          anchor: if h >= zero { "south" } else { "north" },
        )
        content(
          (cx, -0.16),
          text(font: "Inter", size: 7.5pt, fill: brand.muted)[#s.at("label", default: "")],
          anchor: "north",
        )
      }
    })
    #fig-caption(figure)
  ]
}

// —— Stacked: 100% composition, one row per group ——
#let stacked-compose(figure) = {
  let series = figure.at("series", default: ())
  let cats = figure.at("categories", default: ())
  if series.len() == 0 { return }

  let gutter-l = 3.2
  let plot-w = PW - gutter-l - 0.3
  let row-h = 0.92
  let bar-h = 0.56
  let chart-h = series.len() * row-h
  let sc = (lo: 0.0, hi: 100.0, span: 100.0, ticks: (0, 25, 50, 75, 100))

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #cetz.canvas({
      x-grid(gutter-l, 0, plot-w, chart-h, sc, "percent")

      for (si, s) in series.enumerate() {
        let y = chart-h - (si + 0.5) * row-h
        let x = gutter-l
        let shown = s.at("display_values", default: s.values)
        row-label(gutter-l, y, gutter-l, s.at("label", default: ""), size: 7.5pt)

        for (ci, v) in s.values.enumerate() {
          let w = (float(v) / 100.0) * plot-w
          let c = category-palette.at(calc.rem(ci, category-palette.len()))
          rect((x, y - bar-h / 2), (x + w, y + bar-h / 2), fill: c, stroke: 0.4pt + white)
          // Only label slices wide enough to hold the text.
          if w > 0.72 {
            content(
              (x + w / 2, y),
              text(font: "Inter", size: 6.5pt, weight: "semibold", fill: white)[
                #fmt-val(shown.at(ci), "percent")
              ],
            )
          }
          x += w
        }
      }
    })
    #if series.len() == 1 {
      let s = series.first()
      v(5pt)
      for (i, lab) in cats.enumerate() [
        #box[
          #box(
            baseline: 1pt,
            width: 6pt,
            height: 6pt,
            radius: 1pt,
            fill: category-palette.at(calc.rem(i, category-palette.len())),
          )
          #h(3pt)
          #text(font: "Inter", size: 7pt, fill: brand.muted)[
            #lab #fmt-val(s.at("display_values", default: s.values).at(i), "percent")
          ]
        ]
        #h(8pt)
      ]
    } else {
      v(6pt)
      let cells = ()
      cells.push(box(width: 100%, inset: 3pt, fill: brand.grid)[
        #text(font: "Inter", size: 7pt, weight: "semibold", fill: brand.ink)[Category]
      ])
      for s in series {
        cells.push(box(width: 100%, inset: 3pt, fill: brand.grid)[
          #align(right, text(font: "Inter", size: 7pt, weight: "semibold", fill: brand.ink)[#s.label])
        ])
      }
      for (i, lab) in cats.enumerate() {
        cells.push(box[
          #box(
            baseline: 1pt,
            width: 6pt,
            height: 6pt,
            radius: 1pt,
            fill: category-palette.at(calc.rem(i, category-palette.len())),
          )
          #h(3pt)
          #text(font: "Inter", size: 7pt, fill: brand.muted)[#lab]
        ])
        for s in series {
          let shown = s.at("display_values", default: s.values)
          cells.push(align(right, text(font: "Inter", size: 7pt, fill: brand.muted)[
            #fmt-val(shown.at(i), "percent")
          ]))
        }
      }
      grid(
        columns: (2.5fr, ..series.map(_ => 1fr)),
        column-gutter: 8pt,
        row-gutter: 3pt,
        ..cells,
      )
    }
    #fig-caption(figure)
  ]
}

// —— Scatter: one mark per institution, area by enrolment ——
#let scatter-plot(figure) = {
  let pts = figure.at("points", default: ())
  if pts.len() == 0 { return }
  let xu = figure.at("x_unit", default: "")
  let yu = figure.at("y_unit", default: "")

  let xs = pts.map(p => float(p.x))
  let ys = pts.map(p => float(p.y))
  let xlo = calc.min(..xs)
  let xhi = calc.max(..xs)
  let ylo = calc.min(..ys)
  let yhi = calc.max(..ys)
  let xpad = calc.max((xhi - xlo) * 0.12, 0.001)
  let ypad = calc.max((yhi - ylo) * 0.12, 0.001)
  let (bxlo, bxhi) = clamp-bounds(xlo - xpad, xhi + xpad, xu, xlo, xhi)
  let (bylo, byhi) = clamp-bounds(ylo - ypad, yhi + ypad, yu, ylo, yhi)
  let sx = axis-scale(bxlo, bxhi, target: 5)
  let sy = axis-scale(bylo, byhi, target: 4)

  let gutter-l = 1.5
  let plot-w = PW - gutter-l - 0.4
  let chart-h = 5.0
  let px(v) = gutter-l + ((float(v) - sx.lo) / sx.span) * plot-w
  let py(v) = ((float(v) - sy.lo) / sy.span) * chart-h
  let smax = calc.max(..pts.map(p => float(p.at("size", default: 1))), 1.0)

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #cetz.canvas({
      y-grid(gutter-l, 0, plot-w, chart-h, sy, yu)
      x-grid(gutter-l, 0, plot-w, chart-h, sx, xu)

      for p in pts {
        let c = series-fill(p.at("color", default: "peer"))
        let r = 0.09 + 0.17 * calc.sqrt(float(p.at("size", default: 1)) / smax)
        circle(
          (px(p.x), py(p.y)),
          radius: r,
          fill: c.transparentize(45%),
          stroke: 0.5pt + c,
        )
      }
      // Only Marist is named; the rest stay readable as a cloud.
      for p in pts.filter(p => p.at("color", default: "") == "marist") {
        content(
          (px(p.x), py(p.y) + 0.3),
          text(font: "Inter", size: 6.5pt, weight: "bold", fill: brand.marist)[
            #p.at("label", default: "")
          ],
          anchor: "south",
        )
      }

      content(
        (gutter-l + plot-w / 2, -0.52),
        text(font: "Inter", size: 7.5pt, fill: brand.muted)[
          #figure.at("x_label", default: "")
        ],
        anchor: "north",
      )
      content(
        (gutter-l - 0.1, chart-h + 0.18),
        text(font: "Inter", size: 7.5pt, fill: brand.muted)[
          #figure.at("y_label", default: "")
        ],
        anchor: "south-west",
      )
    })
    #if figure.at("legend", default: ()) != () {
      swatch-legend(figure.legend.map(l => (
        label: l.at("label", default: ""),
        color: series-fill(l.at("color", default: "peer")),
      )))
    }
    #fig-caption(figure)
  ]
}

// —— Stats: KPI cards, mirroring the dashboard profile page ——
#let stat-grid(figure) = {
  let items = figure.at("items", default: ())
  let groups = figure.at("groups", default: ())
  if items.len() == 0 and groups.len() == 0 { return }

  let stat-card(it) = block(
    width: 100%,
    inset: 8pt,
    radius: 2pt,
    fill: rgb("#f6f7f8"),
  )[
    #set par(leading: 0.5em, justify: false)
    #text(font: "Inter", size: 7.5pt, fill: brand.muted)[#it.at("label", default: "")]
    #v(3pt)
    #text(font: "Inter", size: 13pt, weight: "bold", fill: brand.ink)[
      #it.at("value", default: "")
    ]
    #if it.at("note", default: "") != "" [
      #v(2pt)
      #text(font: "Inter", size: 7pt, fill: brand.muted)[#it.note]
    ]
  ]

  block(width: 100%, breakable: false)[
    #fig-title(figure)
    #if groups.len() > 0 [
      #grid(
        columns: groups.len(),
        gutter: 10pt,
        ..groups.map(group => block(width: 100%)[
          #text(font: "Inter", size: 8pt, weight: "bold", fill: brand.ink)[
            #group.at("title", default: "")
          ]
          #v(5pt)
          #for (i, it) in group.at("items", default: ()).enumerate() [
            #stat-card(it)
            #if i < group.at("items", default: ()).len() - 1 [#v(6pt)]
          ]
        ]),
      )
    ] else [
      #grid(
        columns: (1fr, 1fr, 1fr),
        gutter: 8pt,
        ..items.map(stat-card),
      )
    ]
    #fig-caption(figure)
  ]
}

#let render-figure(figure) = {
  let kind = figure.at("kind", default: "bar")
  if kind == "scatter" {
    scatter-plot(figure)
  } else if kind == "stats" {
    stat-grid(figure)
  } else if kind == "line" {
    line-trend(figure)
  } else if kind == "bullet" {
    bullet-iqr(figure)
  } else if kind == "bullet-grid" {
    bullet-grid(figure)
  } else if kind == "stacked" {
    stacked-compose(figure)
  } else if kind == "grouped" {
    grouped-bars(figure)
  } else {
    bar-compare(figure)
  }
}
