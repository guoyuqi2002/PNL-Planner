package main

import (
	"sync"
)

// ============================================================
// Session state management.
// Records the "new/import" mode chosen on the first page plus the imported
// shop structure.
// Affects the shop dropdown on the monthly-config page
// (new = defaults, import = what was actually scanned from the Excel file).
// ============================================================

// SessionMode is the session mode
type SessionMode string

const (
	SessionModeNone   SessionMode = ""       // unset (initial)
	SessionModeNew    SessionMode = "new"    // create new
	SessionModeImport SessionMode = "import" // import an existing Excel
)

// SessionState is the session state
type SessionState struct {
	Mode SessionMode `json:"mode"` // new | import

	// Imported files (the two are independent and can be imported at the same time)
	SplitFilePath string `json:"split_file_path"` // split table path (empty = not imported)
	PivotFilePath string `json:"pivot_file_path"` // drill-down table path (empty = not imported)

	// Shop structure scanned from the split table (used in import mode)
	// channel -> []shop (dynamic, from the Excel file)
	ImportedShops map[string][]string `json:"imported_shops"`

	// Department x channel -> mode structure scanned from the split table (dynamic)
	ImportedModes map[string]map[string][]string `json:"imported_modes"`
}

var (
	sessionMutex sync.Mutex
	session      = SessionState{
		Mode:          SessionModeNone,
		ImportedShops: map[string][]string{},
		ImportedModes: map[string]map[string][]string{},
	}
)

// GetSession returns the current session state
func GetSession() SessionState {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	// Return a copy to prevent external mutation
	shops := map[string][]string{}
	for k, v := range session.ImportedShops {
		cp := make([]string, len(v))
		copy(cp, v)
		shops[k] = cp
	}
	modes := map[string]map[string][]string{}
	for k, v := range session.ImportedModes {
		inner := map[string][]string{}
		for k2, v2 := range v {
			cp := make([]string, len(v2))
			copy(cp, v2)
			inner[k2] = cp
		}
		modes[k] = inner
	}
	return SessionState{
		Mode:           session.Mode,
		SplitFilePath:  session.SplitFilePath,
		PivotFilePath:  session.PivotFilePath,
		ImportedShops:  shops,
		ImportedModes:  modes,
	}
}

// StartNewSession starts "new" mode with the default shop/mode structure
func StartNewSession() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	session = SessionState{
		Mode:          SessionModeNew,
		ImportedShops: map[string][]string{},
		ImportedModes: map[string]map[string][]string{},
	}
	// New mode: copy the default structure as the "confirmed shop source"
	for ch, shops := range ChannelShops {
		session.ImportedShops[ch] = append([]string{}, shops...)
	}
	for dept, chModes := range DeptChannelModes {
		session.ImportedModes[dept] = map[string][]string{}
		for ch, ms := range chModes {
			session.ImportedModes[dept][ch] = append([]string{}, ms...)
		}
	}
}

// ImportSplitSession starts "import split table" mode: scans the Excel file to get the
// dynamic shop/mode structure. Returns the scan result + error.
func ImportSplitSession(filePath string) (*ScanResult, error) {
	// Scan the split table (month is a placeholder, not actually needed)
	scan, err := ScanSplitExcel(filePath, "")
	if err != nil {
		return nil, err
	}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	// If no mode has been set yet, set it to import
	if session.Mode == SessionModeNone {
		session.Mode = SessionModeImport
	}

	// Record the file path
	session.SplitFilePath = filePath

	// Reset and fill in the scanned shop/mode structure
	session.ImportedShops = map[string][]string{}
	session.ImportedModes = map[string]map[string][]string{}

	// Extract department x channel -> shop and department x channel -> mode from the scan result
	deptChannelShops := map[string]map[string][]string{} // dept->channel->shops
	for _, c := range scan.Configs {
		// Collect modes
		if session.ImportedModes[c.Department] == nil {
			session.ImportedModes[c.Department] = map[string][]string{}
		}
		ms := session.ImportedModes[c.Department][c.Channel]
		if !contains(ms, c.Mode) {
			session.ImportedModes[c.Department][c.Channel] = append(ms, c.Mode)
		}
		// Collect shops (per channel)
		if deptChannelShops[c.Department] == nil {
			deptChannelShops[c.Department] = map[string][]string{}
		}
		ss := deptChannelShops[c.Department][c.Channel]
		if !contains(ss, c.Shop) {
			deptChannelShops[c.Department][c.Channel] = append(ss, c.Shop)
		}
	}
	// Channel -> shops (merge that channel's shops across all departments, dedup, keep order)
	for _, ch := range Channels {
		seen := map[string]bool{}
		var merged []string
		for _, dept := range Departments {
			if shops, ok := deptChannelShops[dept]; ok {
				for _, s := range shops[ch] {
					if !seen[s] {
						seen[s] = true
						merged = append(merged, s)
					}
				}
			}
		}
		if len(merged) > 0 {
			session.ImportedShops[ch] = merged
		}
	}

	return scan, nil
}

// ImportPivotSession starts "import drill-down table" mode: only records the file path
// (the drill-down table has no shop concept)
func ImportPivotSession(filePath string) error {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	if session.Mode == SessionModeNone {
		session.Mode = SessionModeImport
	}
	session.PivotFilePath = filePath
	return nil
}

// GetDynamicShops returns the dynamic shops (session state first, otherwise defaults)
func GetDynamicShops(channel string) []string {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	if len(session.ImportedShops) > 0 {
		if shops, ok := session.ImportedShops[channel]; ok {
			return shops
		}
		return []string{}
	}
	return ChannelShops[channel]
}

// GetDynamicModes returns the dynamic modes (session state first, otherwise defaults)
func GetDynamicModes(dept, channel string) []string {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	if len(session.ImportedModes) > 0 {
		if dm, ok := session.ImportedModes[dept]; ok {
			if ms, ok := dm[channel]; ok {
				return ms
			}
			return []string{}
		}
		return []string{}
	}
	return GetModes(dept, channel)
}
