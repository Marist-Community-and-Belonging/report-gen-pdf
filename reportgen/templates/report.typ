// Marist / OCB publishable dashboard report — US Letter
#import "plots.typ": render-figure

#let data = json(sys.inputs.at("data", default: "../data.json"))

#let marist-red = rgb("#c91235")
#let greystone = rgb("#63666f")
#let body-ink = rgb("#202127")
#let link-blue = rgb("#005ea8")

#let external-link(url, body) = text(fill: link-blue)[
  #underline[#link(url)[#body]]
]

#let chapters = data.at("chapters", default: ())

#let callout-box(callout) = block(
  width: 100%,
  inset: (left: 10pt, rest: 8pt),
  stroke: (left: 2.5pt + marist-red),
  fill: rgb("#f6f7f8"),
)[
  #if callout.at("label", default: "") != "" [
    #text(size: 9pt, weight: "bold", fill: marist-red)[#callout.label]
    #v(2pt)
  ]
  #text(size: 10pt)[#callout.at("body", default: "")]
]

#set page(
  paper: "us-letter",
  margin: (top: 1.5in, bottom: 1in, left: 1.25in, right: 1.25in),
  header: context {
    if counter(page).get().first() == 1 {
      v(0.5in)
      align(center)[
        #image("../assets/ocb-header.jpg", height: 0.7in)
      ]
    } else {
      set text(font: "Inter", size: 9pt, fill: greystone)
      grid(
        columns: (1fr, 1fr),
        align(left)[#data.at("office", default: "Office of Community and Belonging")],
        align(right)[#data.at("title", default: "")],
      )
      v(4pt)
      line(length: 100%, stroke: 0.5pt + rgb("#d9dae4"))
    }
  },
  header-ascent: 0.3in,
  footer: context {
    set text(font: "Inter", size: 9pt, fill: greystone)
    set par(leading: 12pt)
    line(length: 100%, stroke: 0.5pt + rgb("#d9dae4"))
    v(6pt)
    grid(
      columns: (1fr, auto),
      align(left)[
        #data.at("office", default: "Office of Community and Belonging")
        #if data.at("phone", default: "") != "" [ · #data.phone]
      ],
      align(right)[Page #counter(page).display()],
    )
  },
  footer-descent: 0.35in,
)

#set text(font: "Inter", size: 11pt, fill: body-ink)
#set par(leading: 0.7em, justify: true, spacing: 0.9em)
#set heading(numbering: none)

#show heading.where(level: 1): it => {
  set text(font: "Source Serif Pro", size: 24pt, weight: "regular", fill: body-ink)
  block(above: 0pt, below: 10pt, it.body)
}

#show heading.where(level: 2): it => {
  set text(font: "Source Serif Pro", size: 15pt, weight: "semibold", fill: body-ink)
  block(above: 1.4em, below: 0.55em, breakable: false, it.body)
}

#show heading.where(level: 3): it => {
  set text(font: "Inter", size: 11pt, weight: "bold", fill: body-ink)
  block(above: 1em, below: 0.4em, it.body)
}

// —— Cover / title block ——
#heading(level: 1, outlined: false, data.title)

#if data.at("subtitle", default: "") != "" [
  #text(font: "Inter", size: 13pt, fill: greystone, weight: "semibold")[#data.subtitle]
  #v(6pt)
]

#text(font: "Inter", size: 10pt, fill: greystone)[
  #data.at("date", default: "")
  #if data.at("author", default: "") != "" [ · #data.author]
]

#if data.at("release", default: none) != none [
  #v(4pt)
  #text(font: "Inter", size: 9pt, fill: greystone)[
    IPEDS #data.release.at("collection_year", default: "")
    (#data.release.at("release_type", default: "final"))
    #if data.release.at("note", default: "") != "" [ — #data.release.note]
  ]
]

#v(8pt)
#line(length: 100%, stroke: 1.5pt + marist-red)
#v(14pt)

#if data.at("intro", default: "") != "" [
  #text(font: "Inter", size: 11pt, weight: "semibold")[#data.intro]
  #v(12pt)
]

#outline(
  title: [#text(font: "Source Serif Pro", size: 15pt, weight: "semibold")[Contents]],
  depth: 2,
  indent: 0pt,
)

#let date-guide = data.at("date_guide", default: ())
#if date-guide.len() > 0 [
  #v(8pt)
  #text(font: "Source Serif Pro", size: 12pt, weight: "semibold")[Data guide]
  #v(3pt)
  #table(
    columns: (1.15fr, 2.85fr),
    inset: (x: 4pt, y: 2pt),
    stroke: none,
    ..date-guide.map(item => (
      [#text(size: 8pt, weight: "semibold")[#item.term]],
      [#set par(justify: false)
       #text(size: 8pt, fill: greystone)[#item.meaning]],
    )).flatten(),
  )
]

#let comparison-groups = data.at("comparison_groups", default: ())
#if comparison-groups.len() > 0 [
  #v(8pt)
  #text(font: "Source Serif Pro", size: 12pt, weight: "semibold")[Comparison groups]
  #v(3pt)
  #table(
    columns: (1.2fr, 1.45fr, 3.35fr),
    inset: (x: 5pt, y: 4pt),
    stroke: (x: none, y: 0.5pt + rgb("#d9dae4")),
    fill: (x, y) => if y == 0 { rgb("#f1f2f4") } else { white },
    table.header(
      [#text(size: 8pt, weight: "bold")[Group]],
      [#text(size: 8pt, weight: "bold")[Role]],
      [#text(size: 8pt, weight: "bold")[Institutions]],
    ),
    ..comparison-groups.map(group => (
      [#text(size: 7.5pt, weight: "semibold")[#group.label]],
      [#set par(justify: false)
       #text(size: 7.5pt, fill: greystone)[#group.role]],
      [#set par(justify: false)
       #text(size: 7.5pt, fill: greystone)[#group.institutions]],
    )).flatten(),
  )
  #if data.at("comparison_context_url", default: "") != "" [
    #v(3pt)
    #text(size: 8pt, fill: greystone)[Neither group is a ranking or prediction. #external-link(data.comparison_context_url)[See Marist's official peer and aspirant groups].]
  ]
]

// —— Chapters ——
#for chapter in chapters [
  #pagebreak(weak: true)
  #let figures = chapter.at("figures", default: ())
  #block(breakable: false)[
    #if chapter.at("anchor", default: "") != "" [
      #heading(level: 2, chapter.heading)#label(chapter.anchor)
    ] else [
      #heading(level: 2, chapter.heading)
    ]

    #if chapter.at("body", default: "") != "" [
      #par[#chapter.body]
    ]

    #if chapter.at("bullets", default: ()) != () [
      #v(4pt)
      #for item in chapter.bullets [
        - #item
      ]
    ]

    #if figures.len() > 0 [
      #v(10pt)
      #render-figure(figures.first())
    ]
  ]

  #if figures.len() > 1 [
    #for fig in figures.slice(1) [
      #v(10pt)
      #render-figure(fig)
    ]
  ]

  #let callouts = chapter.at("callouts", default: ())
  #if callouts.len() > 0 [
    #v(8pt)
    #grid(
      columns: if callouts.len() > 1 { 2 } else { 1 },
      gutter: 10pt,
      ..callouts.map(callout-box),
    )
  ]
]

#let references = data.at("references", default: ())
#if references.len() > 0 [
  == References
  #if data.at("reference_note", default: "") != "" [
    #text(size: 9pt, fill: greystone)[#data.reference_note]
    #v(6pt)
  ]
  #table(
    columns: (1.25fr, 2.15fr, 2.4fr),
    inset: (x: 5pt, y: 4pt),
    stroke: (x: none, y: 0.5pt + rgb("#d9dae4")),
    fill: (x, y) => if y == 0 { rgb("#f1f2f4") } else { white },
    table.header(
      [#text(size: 8pt, weight: "bold")[Section]],
      [#text(size: 8pt, weight: "bold")[IPEDS component / variable]],
      [#text(size: 8pt, weight: "bold")[Measures]],
    ),
    ..references.map(ref => (
      [#text(size: 8pt, weight: "semibold")[#link(label(ref.anchor))[#ref.section]]],
      [#set par(justify: false)
       #text(size: 7.5pt, fill: greystone)[#ref.source]],
      [#set par(justify: false)
       #text(size: 7.5pt, fill: greystone)[#ref.coverage]],
    )).flatten(),
  )
  #if data.at("source_url", default: "") != "" [
    #v(6pt)
    #text(size: 9pt, fill: greystone)[Official source: #external-link(data.source_url)[NCES/IPEDS Use the Data].]
  ]
]
