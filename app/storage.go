package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ============================================================
// Data structures
// ============================================================

// ConfigRow is one SKU record of the monthly config (temporary data, kept in memory)
type ConfigRow struct {
	YearMonth            string    `json:"year_month"`
	Department           string    `json:"department"`
	Channel              string    `json:"channel"`
	Mode                 string    `json:"mode"`
	Shop                 string    `json:"shop"`
	MonthlyRevenueTarget float64   `json:"monthly_revenue_target"`
	TargetMode           string    `json:"target_mode"`
	ComboName            string    `json:"combo_name"`
	RevenueShare         float64   `json:"revenue_share"`
	SubItems             []SubItem `json:"sub_items"`
	PlatformPrice        float64   `json:"platform_price"`
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

// SkuLibraryItem is one entry of the SKU library (persistent, stored in a local file)
type SkuLibraryItem struct {
	SkuName  string    `json:"sku_name"`
	SubItems []SubItem `json:"sub_items"`
}

// ============================================================
// SKU library persistence (stored in %APPDATA%/pnl_planner/sku_library.json)
// ============================================================

var skuLibraryFile string

func initStorage() {
	dir := appDataDir()
	os.MkdirAll(dir, 0755)
	skuLibraryFile = filepath.Join(dir, "sku_library.json")
}

func loadSkuLibrary() []SkuLibraryItem {
	initStorage()
	data, err := os.ReadFile(skuLibraryFile)
	if err != nil {
		return []SkuLibraryItem{}
	}
	items := []SkuLibraryItem{}
	json.Unmarshal(data, &items)
	return items
}

func saveSkuLibrary(items []SkuLibraryItem) error {
	initStorage()
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(skuLibraryFile, data, 0644)
}

// SaveSkuToLibrary creates/updates a SKU (upsert)
func SaveSkuToLibrary(name string, subItems []SubItem) error {
	items := loadSkuLibrary()
	// Check whether it already exists
	found := false
	for i, it := range items {
		if it.SkuName == name {
			items[i].SubItems = subItems
			found = true
			break
		}
	}
	if !found {
		items = append(items, SkuLibraryItem{SkuName: name, SubItems: subItems})
	}
	return saveSkuLibrary(items)
}

func DeleteSkuFromLibrary(name string) error {
	items := loadSkuLibrary()
	result := []SkuLibraryItem{}
	for _, it := range items {
		if it.SkuName != name {
			result = append(result, it)
		}
	}
	return saveSkuLibrary(result)
}

func GetSkuFromLibrary(name string) *SkuLibraryItem {
	items := loadSkuLibrary()
	for _, it := range items {
		if it.SkuName == name {
			return &it
		}
	}
	return nil
}

// ============================================================
// Monthly config (temporary data, kept in memory, cleared on restart)
// ============================================================

var (
	configMutex sync.Mutex
	configStore = []ConfigRow{} // monthly config in memory, cleared when the program restarts
)

// SaveConfigRows saves the config of a dimension (delete first, then save)
func SaveConfigRows(yearMonth, dept, channel, mode, shop string,
	monthlyTarget float64, targetMode string, rows []ConfigRow) int {
	configMutex.Lock()
	defer configMutex.Unlock()
	// Delete old data for this dimension first
	filtered := []ConfigRow{}
	for _, r := range configStore {
		if !(r.YearMonth == yearMonth && r.Department == dept &&
			r.Channel == channel && r.Mode == mode && r.Shop == shop) {
			filtered = append(filtered, r)
		}
	}
	configStore = filtered
	// Append the new data
	saved := 0
	for _, r := range rows {
		r.YearMonth = yearMonth
		r.Department = dept
		r.Channel = channel
		r.Mode = mode
		r.Shop = shop
		r.MonthlyRevenueTarget = monthlyTarget
		if targetMode == "" {
			targetMode = "company"
		}
		r.TargetMode = targetMode
		configStore = append(configStore, r)
		saved++
	}
	return saved
}

// QueryByDimension queries the config of a dimension
func QueryByDimension(yearMonth, dept, channel, mode, shop string) []ConfigRow {
	configMutex.Lock()
	defer configMutex.Unlock()
	result := []ConfigRow{}
	for _, r := range configStore {
		if r.YearMonth == yearMonth && r.Department == dept &&
			r.Channel == channel && r.Mode == mode && r.Shop == shop {
			result = append(result, r)
		}
	}
	return result
}

// QueryByMonth queries all configs of a month
func QueryByMonth(yearMonth string) []ConfigRow {
	configMutex.Lock()
	defer configMutex.Unlock()
	result := []ConfigRow{}
	for _, r := range configStore {
		if r.YearMonth == yearMonth {
			result = append(result, r)
		}
	}
	return result
}

// QueryByDepartment queries all monthly configs of a department
func QueryByDepartment(dept string) []ConfigRow {
	configMutex.Lock()
	defer configMutex.Unlock()
	result := []ConfigRow{}
	for _, r := range configStore {
		if r.Department == dept {
			result = append(result, r)
		}
	}
	return result
}

// GetSavedDimensions queries all saved dimensions of a month (deduplicated)
type DimensionInfo struct {
	Department           string  `json:"department"`
	Channel              string  `json:"channel"`
	Mode                 string  `json:"mode"`
	Shop                 string  `json:"shop"`
	MonthlyRevenueTarget float64 `json:"monthly_revenue_target"`
	TargetMode           string  `json:"target_mode"`
}

func GetSavedDimensions(yearMonth string) []DimensionInfo {
	configMutex.Lock()
	defer configMutex.Unlock()
	seen := map[string]bool{}
	result := []DimensionInfo{}
	for _, r := range configStore {
		if r.YearMonth != yearMonth {
			continue
		}
		key := r.Department + "|" + r.Channel + "|" + r.Mode + "|" + r.Shop
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, DimensionInfo{
			Department: r.Department, Channel: r.Channel,
			Mode: r.Mode, Shop: r.Shop,
			MonthlyRevenueTarget: r.MonthlyRevenueTarget,
			TargetMode:           r.TargetMode,
		})
	}
	return result
}

// DeleteByDimension deletes all configs of a dimension
func DeleteByDimension(yearMonth, dept, channel, mode, shop string) {
	configMutex.Lock()
	defer configMutex.Unlock()
	filtered := []ConfigRow{}
	for _, r := range configStore {
		if !(r.YearMonth == yearMonth && r.Department == dept &&
			r.Channel == channel && r.Mode == mode && r.Shop == shop) {
			filtered = append(filtered, r)
		}
	}
	configStore = filtered
}

// ClearConfig clears all monthly configs (for debugging)
func ClearConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()
	configStore = []ConfigRow{}
}
