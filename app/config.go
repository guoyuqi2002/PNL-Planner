package main

// Dimension configuration constants (mirrors the original Python config.py)

// Departments
var Departments = []string{"Influencer Distribution", "Innovation & Custom", "Channel Operations"}

// PivotTopBlockName maps a department to the top-level summary block name in the
// drill-down table (the English department name is used as-is)
func PivotTopBlockName(department string) string {
	return department
}

// PivotChannelBlocks returns the channel block order of a department in the drill-down
// table (only channels that actually have config for the department, ordered to match the
// template: Douyin / Kuaishou / WeChat Channels / Xiaohongshu)
func PivotChannelBlocks(department string) []string {
	var result []string
	modes, ok := DeptChannelModes[department]
	if !ok {
		return []string{}
	}
	// Output the department's channels in the fixed channel order
	for _, ch := range Channels {
		if ms, has := modes[ch]; has && len(ms) > 0 {
			result = append(result, ch)
		}
	}
	return result
}

// Channels (fixed order, matching the template)
var Channels = []string{"Douyin", "Kuaishou", "WeChat Channels", "Xiaohongshu"}

// Department x channel -> mode list
var DeptChannelModes = map[string]map[string][]string{
	"Influencer Distribution": {
		"Douyin":          {"Influencer Live"},
		"Kuaishou":        {"Influencer Live"},
		"WeChat Channels": {"Influencer Live"},
		"Xiaohongshu": {}, // Influencer Distribution has no Xiaohongshu
	},
	"Innovation & Custom": {
		"Douyin":          {"Influencer Short Video", "Custom"},
		"Kuaishou":        {"Influencer Short Video", "Custom"},
		"WeChat Channels": {"Influencer Short Video", "Custom"},
		"Xiaohongshu":     {"Influencer Post", "Custom"},
	},
	"Channel Operations": {
		"Douyin":          {"DP", "Product Card"},
		"Kuaishou":        {"KP", "Product Card"},
		"WeChat Channels": {"SP", "Product Card"},
		"Xiaohongshu":     {"XP", "Owned Post", "Product Card"},
	},
}

// Channel -> shop list (neutral demo shop names)
var ChannelShops = map[string][]string{
	"Douyin":          {"Demo Fresh Flagship Store", "Demo Official Flagship Store", "Demo Instant-Food Flagship Store", "Demo Selection Flagship Store"},
	"Kuaishou":        {"Demo Fresh Flagship Store", "Demo Official Flagship Store"},
	"WeChat Channels": {"Demo Mini Store", "Demo Selection Mini Store"},
	"Xiaohongshu":     {"Demo Flagship Store"},
}

// Block titles (department -> mode -> title text)
var BlockTitles = map[string]map[string]string{
	"Influencer Distribution": {
		"Influencer Live": "Influencer Distribution - Influencer Live",
	},
	"Innovation & Custom": {
		"Influencer Short Video": "Innovation & Custom - Influencer Short Video",
		"Influencer Post":        "Innovation & Custom - Influencer Post",
		"Custom":                 "Innovation & Custom - Custom",
	},
	"Channel Operations": {
		"DP":          "Channel Operations - DP",
		"KP":          "Channel Operations - KP",
		"SP":          "Channel Operations - SP",
		"XP":          "Channel Operations - XP",
		"Product Card": "Channel Operations - Product Card",
		"Owned Post":  "Channel Operations - Owned Post & Product Card", // Xiaohongshu owned post + product card merged
	},
}

// Fiscal-year months (starting in July)
var MonthsFY = []string{"Jul", "Aug", "Sep", "Oct", "Nov", "Dec", "Jan", "Feb", "Mar", "Apr", "May", "Jun"}

// Fiscal-year months (YYYY-MM format)
func AllMonthsFY2026() []string {
	return []string{"2026-07", "2026-08", "2026-09", "2026-10", "2026-11", "2026-12",
		"2027-01", "2027-02", "2027-03", "2027-04", "2027-05", "2027-06"}
}

// GetModes returns the modes of a department on a channel
func GetModes(dept, channel string) []string {
	if ch, ok := DeptChannelModes[dept]; ok {
		return ch[channel]
	}
	return nil
}

// GetShops returns the shops of a channel
func GetShops(channel string) []string {
	return ChannelShops[channel]
}

// GetBlockTitle returns the block title
func GetBlockTitle(dept, mode string) string {
	if m, ok := BlockTitles[dept]; ok {
		if t, ok := m[mode]; ok {
			return t
		}
	}
	return dept + "-" + mode
}

// Split table detail columns
var SplitCols = []string{
	"Shop", "Product Name", "Platform Revenue (GSV)", "Company Revenue (zpr9 excl. tax)", "Standard Cost",
	"Discount Amount", "Discount Rate", "Traffic Spend", "Influencer Commission", "Agency Commission",
	"Influencer Slot Fee", "Service Fee", "Non-HQ Support", "AOV (zpr9 excl. tax)",
	"Order Qty", "Standard Cost Total", "Gross Margin",
}

// Summary table columns
var SummaryCols = []string{
	"Channel", "Segment", "Platform Revenue (GSV)", "Company Revenue (zpr9 excl. tax)", "Standard Cost", "Gross Margin",
	"TP1.2 E-commerce Service Fee", "TP2.2 Platform Deduction", "CP1.2 Sales Discount", "CP5.1 Platform Traffic",
	"CP5.2 Influencer Resource", "CP5.3 Other Traffic", "Non-HQ Support", "AP (Rate)", "TP (Rate)", "CP (Rate)",
	"PP (Rate)", "MP (Rate)", "FP (Rate)", "Non-HQ Support (Rate)",
	"Est. P&L (before Non-HQ)", "Est. P&L (after Non-HQ)",
}

// Drill-down table fee-item rows
var PivotRows = []string{
	"Revenue", "Gross Profit", "AP", "TP", "CP", "PP", "MP", "FP",
	"Assessed Net Profit", "Non-HQ Support", "Operating Net Profit",
}
