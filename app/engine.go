package main

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

// ============================================================
// Master data (master_data.json, embedded)
// ============================================================

//go:embed master_data.json
var masterDataBytes []byte

// Material is a material master-data record
type Material struct {
	MatID            string   `json:"mat_id"`
	Name             string   `json:"name"`
	RetailSpec       float64  `json:"retail_spec"`
	StdCost          *float64 `json:"std_cost"`
	DistLandingPrice *float64 `json:"dist_landing_price"`
	EcAssessRevenue  *float64 `json:"ec_assess_revenue"`
}

var masterCache map[string]Material

// LoadMaster loads the master data (external override file first, then the embedded copy;
// the JSON is parsed on first call)
func LoadMaster() map[string]Material {
	if masterCache != nil {
		return masterCache
	}
	m := make(map[string]Material)
	// External override (written by the database-update feature) takes priority
	if data, err := os.ReadFile(masterOverridePath()); err == nil {
		if err := json.Unmarshal(data, &m); err == nil && len(m) > 0 {
			masterCache = m
			return m
		}
	}
	json.Unmarshal(masterDataBytes, &m)
	masterCache = m
	return m
}

func getSpec(matID string) float64 {
	m := LoadMaster()
	if mat, ok := m[matID]; ok {
		return mat.RetailSpec
	}
	return 0
}

func getAssess(matID string) float64 {
	m := LoadMaster()
	if mat, ok := m[matID]; ok {
		if mat.EcAssessRevenue != nil {
			return *mat.EcAssessRevenue
		}
	}
	return 0
}

func getCost(matID string) float64 {
	m := LoadMaster()
	if mat, ok := m[matID]; ok {
		if mat.StdCost != nil {
			return *mat.StdCost
		}
	}
	return 0
}

// getComboUnitCost computes the unit standard cost of a SKU combo (per bag/piece)
// Multiple materials: Sum(sub-material standard cost x bags) / total bags? No.
// Actual: E is the "cost per order unit" and P = E x O (total cost)
// Total standard cost StdCostTotal = Sum(cost/spec x bags) x order qty
// So E = StdCostTotal / order qty = Sum(cost/spec x bags) <- unit cost
func getComboUnitCost(subItems []SubItem) float64 {
	var total float64
	for _, it := range subItems {
		spec := getSpec(it.MatID)
		cost := getCost(it.MatID)
		if spec > 0 {
			total += cost / spec * float64(it.Bags)
		}
	}
	return total
}

// ============================================================
// Compute engine
// ============================================================

// SubItem is a sub-material of a SKU combo
type SubItem struct {
	MatID string `json:"mat_id"`
	Name  string `json:"name"`
	Bags  int    `json:"bags"`
}

// ComputeInput is the computation input
type ComputeInput struct {
	SubItems             []SubItem `json:"sub_items"`
	PlatformPrice        float64   `json:"platform_price"`
	MonthlyRevenueTarget float64   `json:"monthly_revenue_target"`
	TargetMode           string    `json:"target_mode"` // "company" (default) or "platform" (by gross GSV platform revenue)
	RevenueShare         float64   `json:"revenue_share"`
	ApRate               float64   `json:"ap_rate"`
	MpRate               float64   `json:"mp_rate"`
	TrafficCost          float64   `json:"traffic_cost"`
	Tp12Rate             *float64  `json:"tp12_rate"`
	Cp52Rate             float64   `json:"cp52_rate"`
	Cp52Slot             float64   `json:"cp52_slot"`
	Cp53Rate             float64   `json:"cp53_rate"`
	LogisticsCost        float64   `json:"logistics_cost"`
	FpRate               float64   `json:"fp_rate"`
}

// ComputeResult is the computation result
type ComputeResult struct {
	EstQty           float64 `json:"est_qty"`
	PlatformRevUntax float64 `json:"platform_rev_untax"`
	PlatformRevGsv   float64 `json:"platform_rev_gsv"`
	CompanyRev       float64 `json:"company_rev"`
	StdCostTotal     float64 `json:"std_cost_total"`
	GrossProfit      float64 `json:"gross_profit"`
	GrossRate        float64 `json:"gross_rate"`
	Ap               float64 `json:"ap"`
	ApRate           float64 `json:"ap_rate"`
	Tp12             float64 `json:"tp12"`
	Tp22             float64 `json:"tp22"`
	TpTotal          float64 `json:"tp_total"`
	Cp12             float64 `json:"cp12"`
	Cp51             float64 `json:"cp51"`
	Cp52             float64 `json:"cp52"`
	Cp53             float64 `json:"cp53"`
	CpTotal          float64 `json:"cp_total"`
	Pp               float64 `json:"pp"`
	Mp               float64 `json:"mp"`
	MpRate           float64 `json:"mp_rate"`
	MarketingFee     float64 `json:"marketing_fee"`
	AssessNet        float64 `json:"assess_net"`
	Fp               float64 `json:"fp"`
	AE               float64 `json:"ae"` // AE item = company revenue x 1.08%
	NonSupport       float64 `json:"non_support"`
	OpNet            float64 `json:"op_net"`
	DiscountRate     float64 `json:"discount_rate"`
	Aov              float64 `json:"aov"`
}

// ComputeSKU computes all metrics for one SKU (a 1:1 port of the Python compute_sku)
func ComputeSKU(in ComputeInput) ComputeResult {
	// Step 1: sums Sum(assess revenue/spec x bags), Sum(standard cost/spec x bags)
	var sumAssess, sumCost float64
	for _, it := range in.SubItems {
		spec := getSpec(it.MatID)
		assess := getAssess(it.MatID)
		cost := getCost(it.MatID)
		var perUnitAssess, perUnitCost float64
		if spec > 0 {
			perUnitAssess = assess / spec
			perUnitCost = cost / spec
		}
		sumAssess += perUnitAssess * float64(it.Bags)
		sumCost += perUnitCost * float64(it.Bags)
	}

	// Step 2: estimated order quantity
	var estQty float64
	if in.TargetMode == "platform" {
		// By platform revenue (gross GSV): estQty = GSV target x share / selling price
		if in.PlatformPrice > 0 {
			estQty = in.MonthlyRevenueTarget * in.RevenueShare / in.PlatformPrice
		}
	} else {
		// By company revenue (excl. tax): estQty = company target x share / Sumassess x 1.13
		if sumAssess > 0 {
			estQty = in.MonthlyRevenueTarget * in.RevenueShare / sumAssess * 1.13
		}
	}

	// Step 3: revenue metrics
	platformRevUntax := in.PlatformPrice * estQty / 1.13
	platformRevGsv := in.PlatformPrice * estQty
	companyRev := sumAssess * estQty / 1.13
	stdCostTotal := sumCost * estQty
	grossProfit := companyRev - stdCostTotal
	var grossRate float64
	if companyRev != 0 {
		grossRate = grossProfit / companyRev
	}

	// Step 4: fees
	ap := companyRev * in.ApRate
	var tp12Rate float64
	if in.Tp12Rate != nil {
		tp12Rate = *in.Tp12Rate
	}
	tp12 := in.PlatformPrice * estQty * tp12Rate
	tp22 := in.PlatformPrice * 0.02 * estQty
	tpTotal := tp12 + tp22
	cp12 := companyRev - platformRevUntax
	cp51 := in.TrafficCost * estQty
	cp52 := in.PlatformPrice*estQty*in.Cp52Rate + in.Cp52Slot
	cp53 := in.PlatformPrice * estQty * in.Cp53Rate
	cpTotal := cp12 + cp51 + cp52 + cp53
	pp := estQty * in.LogisticsCost
	mp := companyRev * in.MpRate

	// Step 5: profit
	marketingFee := tpTotal + cpTotal + ap
	fp := grossProfit * 0.13 * 0.12                             // FP = gross profit x 0.13 x 0.12 (automatic, no manual input)
	ae := companyRev * 0.0108                                   // AE = company revenue x 1.08%
	assessNet := grossProfit - marketingFee - pp - mp - fp - ae // Assessed net = gross - marketing - PP - MP - FP - AE
	opNet := assessNet * 0.75                                   // After-tax net = assessed net x 0.75

	var discountRate float64
	if companyRev != 0 {
		discountRate = cp12 / companyRev
	}
	var aov float64
	if estQty != 0 {
		aov = companyRev / estQty
	}

	return ComputeResult{
		EstQty: estQty, PlatformRevUntax: platformRevUntax, PlatformRevGsv: platformRevGsv,
		CompanyRev: companyRev, StdCostTotal: stdCostTotal,
		GrossProfit: grossProfit, GrossRate: grossRate,
		Ap: ap, ApRate: in.ApRate,
		Tp12: tp12, Tp22: tp22, TpTotal: tpTotal,
		Cp12: cp12, Cp51: cp51, Cp52: cp52, Cp53: cp53, CpTotal: cpTotal,
		Pp: pp, Mp: mp, MpRate: in.MpRate,
		MarketingFee: marketingFee, AssessNet: assessNet, Fp: fp, AE: ae,
		NonSupport: 0, OpNet: opNet,
		DiscountRate: discountRate, Aov: aov,
	}
}

// Aggregate summarizes multiple records (a port of the Python aggregate)
func Aggregate(records []ComputeResult) map[string]float64 {
	agg := map[string]float64{}
	keys := []string{
		"platform_rev_gsv", "company_rev", "std_cost_total", "gross_profit",
		"cp12", "cp51", "cp52", "cp53", "tp12", "tp22", "pp", "mp", "ap",
		"assess_net", "est_qty", "fp", "ae",
	}
	for _, k := range keys {
		agg[k] = 0
	}
	for _, r := range records {
		agg["platform_rev_gsv"] += r.PlatformRevGsv
		agg["company_rev"] += r.CompanyRev
		agg["std_cost_total"] += r.StdCostTotal
		agg["gross_profit"] += r.GrossProfit
		agg["cp12"] += r.Cp12
		agg["cp51"] += r.Cp51
		agg["cp52"] += r.Cp52
		agg["cp53"] += r.Cp53
		agg["tp12"] += r.Tp12
		agg["tp22"] += r.Tp22
		agg["pp"] += r.Pp
		agg["mp"] += r.Mp
		agg["ap"] += r.Ap
		agg["assess_net"] += r.AssessNet
		agg["est_qty"] += r.EstQty
		agg["fp"] += r.Fp
		agg["ae"] += r.AE
	}
	// Derived metrics
	cr := agg["company_rev"]
	if cr != 0 {
		agg["gross_rate"] = agg["gross_profit"] / cr
		agg["discount_rate"] = agg["cp12"] / cr
	}
	agg["tp_total"] = agg["tp12"] + agg["tp22"]
	agg["cp_total"] = agg["cp12"] + agg["cp51"] + agg["cp52"] + agg["cp53"]
	agg["marketing_fee"] = agg["tp_total"] + agg["cp_total"] + agg["ap"]
	if agg["platform_rev_gsv"] != 0 {
		agg["platform_rev_untax"] = agg["platform_rev_gsv"] / 1.13
	}
	if agg["est_qty"] != 0 {
		agg["aov"] = cr / agg["est_qty"]
	}
	agg["op_net"] = agg["assess_net"] * 0.75
	return agg
}

// computeFromRow computes from a db record (adapts the storage layer's ConfigRow)
func computeFromRow(row ConfigRow) ComputeResult {
	return ComputeSKU(ComputeInput{
		SubItems:             row.SubItems,
		PlatformPrice:        row.PlatformPrice,
		MonthlyRevenueTarget: row.MonthlyRevenueTarget,
		TargetMode:           row.TargetMode,
		RevenueShare:         row.RevenueShare,
		ApRate:               row.ApRate,
		MpRate:               row.MpRate,
		TrafficCost:          row.TrafficCost,
		Tp12Rate:             row.Tp12Rate,
		Cp52Rate:             row.Cp52Rate,
		Cp52Slot:             row.Cp52Slot,
		Cp53Rate:             row.Cp53Rate,
		LogisticsCost:        row.LogisticsCost,
	})
}

// User data directory (stores the SKU library and other persistent data).
// A separate demo profile keeps synthetic runs isolated from existing user data.
func appDataDir() string {
	if d := os.Getenv("PNL_PLANNER_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("PNL_PLANNER_TEST_DATA_DIR"); d != "" {
		return d
	}
	dir, _ := os.UserConfigDir() // Windows: %APPDATA%
	return filepath.Join(dir, "pnl_planner")
}
