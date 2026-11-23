# Channel P&L Planner

An offline desktop planning & settlement tool for e-commerce influencer channels, built
during an internship with the direct-sales team of a Chinese FMCG company. It replaced a
set of error-prone manual Excel workflows with a small app that turns a product master
database plus monthly targets into split/drill-down budget workbooks, and turns raw
channel order exports into per-SKU settlement tables and weekly P&L reports.

## Two modes

### Planning mode

`SKU Library -> Monthly Config -> Export`

- **SKU Library** - build combo SKUs from the material master data (a SKU = several
  sub-materials x bag counts). Saving is blocked unless every sub-material has spec,
  e-commerce assess revenue and standard cost, with a detailed missing-data report.
- **Monthly Config** - enter budget parameters per dimension
  `month x department x channel x mode x shop` (e.g. Douyin / Kuaishou / WeChat Channels
  / Xiaohongshu), with revenue-share, platform price and all fee rates, plus a live
  P&L preview per SKU (debounced, computed by the same Go engine used for export).
  The monthly target can be expressed as company revenue (excl. tax) or platform GSV.
- **Compute engine** (`engine.go`) - for each SKU: Σassess/Σcost from master data,
  estimated order quantity from the target mode, revenue/cost breakdown, then the fee
  stack (AP, TP1.2/TP2.2, CP1.2/CP5.1-5.3, PP, MP, FP = gross x 0.13 x 0.12,
  AE = revenue x 1.08%), assessed net profit and after-tax net (x 0.75).
- **Export** (`exporter.go`) - generates a 5-sheet **split table** (summary + one sheet
  per channel with department/mode blocks, shop rows, live SUM/ratio Excel formulas and
  a FullCalcOnLoad flag so formulas recalc on open) and a **drill-down table** per
  department (blocks = department total + its channels; rows = 11 fee items; columns =
  fiscal-year total + 12 months x amount/rate). Import/fill modes can also scan an
  existing workbook and fill values into the right cells without touching its formulas.

### Settlement mode

`order files -> parse/filter -> threshold merge -> settlement table / weekly report`

- Ingests raw order exports from three channels: **WeChat Channels** (.xlsx, columns
  located by header name to survive layout shifts), **Kuaishou** (password-encrypted
  .xlsx, password entered at runtime), **Douyin** (.csv).
- Row-level filters per channel: order status, refunds, traffic source (keep only
  influencer-driven orders such as the influencer marketplace / distribution), order
  carrier, and own-brand accounts.
- SKU codes are normalized through an alias table (`display_name`) so the same product
  sold under different channel-specific codes is grouped together; commission is
  accumulated row-by-row where the export provides a rate column.
- **Threshold merge**: SKUs whose order count falls below the threshold are folded into
  the most-ordered SKU - revenue/commission keep their real values, while company
  revenue for the merged slice is derived through the group's modal unit price (ideal
  discount rate).
- The editable result grid (logistics, commission rates, slot fees, AP/MP) exports an
  18-column **settlement table** per SKU or a **weekly report** with P&L amount/rate and
  a fee breakdown.

### Data update tab

Master data, barcode->material combo mapping, alias merging and per-group "master code"
selection can all be updated in-app from xlsx/xlsm/csv files. Updates are written to
`%APPDATA%/pnl_planner/` as JSON overrides that take priority over the embedded copies,
apply immediately, and can be reverted by deleting the file. Excel-style `="0123..."`
code cells are sanitized on import.

## Tech stack

- **Backend**: Go 1.25, [Wails v2](https://wails.io) (desktop shell), `excelize/v2`
  for Excel I/O including encrypted-workbook support, `golang.org/x/text` for
  GBK-encoded CSV decoding
- **Frontend**: Vue 3 (`<script setup>`), Element Plus, Vite
- **Data**: material master data and code mappings embedded via `go:embed`; user SKU
  library and overrides stored locally under `%APPDATA%/pnl_planner/`; monthly config
  held in memory (exports are its archive)
- Fully offline; no telemetry, no network calls

## Repository layout

```
app/                      Wails desktop application
  *.go                    backend: app bindings, config, compute engine, importer,
                          exporter, settlement (jiean), session, storage, updater
  master_data.json        SYNTHETIC demo material master data (embedded)
  code_mapping.json       SYNTHETIC demo barcode->material mapping (embedded)
  frontend/               Vue 3 + Element Plus UI
  build/                  Wails build assets (icons, manifests, installer)
examples/                 Synthetic order fixture for the settlement workflow

```

## Build & run

Prerequisites: Go 1.25+, Node.js + npm, the [Wails CLI](https://wails.io/docs/gettingstarted/installation)
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`), and WebView2 on Windows.

```bash
cd app
wails dev     # hot-reload development
wails build   # production build -> build/bin/
```

The frontend can also be developed standalone: `cd app/frontend && npm install && npm run dev`.

## Data note

`app/master_data.json` and `app/code_mapping.json` are **synthetic demo datasets**
(18 generic "Demo ..." food products and 15 fake EAN-style barcodes with a few alias
groups). The company's real internal product database, order files and workbooks are
**not** included in this repository. All company, shop and product names in the code
are neutral placeholders.



## Synthetic portfolio walkthrough

Set PNL_PLANNER_DATA_DIR to the path of a new empty local directory before launching Wails to isolate the demo catalog, SKU library and overrides from an existing profile.

- **Planning** — Add a SKU based on demo material 1000000001 and two units. In **Monthly Config**, choose August 2026, Influencer Distribution, Douyin, Influencer Live and Demo Fresh Flagship Store. Select platform revenue as the target basis. Enter a target of 1,000, allocate 100% to the SKU and use a platform price of 40. The preview shows 25 estimated units and 275 in standard cost. Save the plan and generate the split workbook in **Export**.
- **Settlement** — Select Douyin and import examples/demo-douyin-orders.csv. Set the quantity threshold to 5. The importer keeps three order rows and merges the smaller SKU into Demo Wonton Twin Pack. The output shows 320 in platform revenue, including 45 from the merged SKU. Review the fee inputs and export the settlement workbook.
- **Data Update** — Use the demo profile for imports that change materials or SKU mappings.

The order fixture uses fictional order references and creators. Both workflows run without an API key.
