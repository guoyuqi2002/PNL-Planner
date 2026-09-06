// Chanmama live-selling influencer ranking scraper module.
// Data directory: set the CMM_SCRAPER_DIR environment variable (default "./out").
// Writes incremental results to <dir>/ranking.json.
// Note: chanmama.com renders amounts with "w" or the Chinese "wan" (both mean 10k);
// the "wan" literal is unicode-escaped to keep this file ASCII-only.
import fs from "node:fs";

const DIR = (process.env.CMM_SCRAPER_DIR || "./out").replace(/\/+$/, "") + "/";
const RES = DIR + "ranking.json";

export async function getChanTab(browser) {
  const tabs = await browser.tabs.list();
  const info = tabs.find(t => t.url && t.url.includes("chanmama"));
  return info ? await browser.tabs.get(info.id) : await browser.tabs.new();
}

async function extractPage(tab) {
  const cells = await tab.playwright.locator("table td").allTextContents();
  const out = [];
  for (let i = 0; i + 8 < cells.length; i += 9) {
    const c = cells.slice(i, i + 9);
    const np = (c[1] || "").trim().split(/\s+/).filter(x => x);
    let name = "", douyin = "";
    if (np.length >= 2) { name = np.slice(0, -1).join(""); douyin = np[np.length - 1]; }
    else if (np.length === 1) { name = np[0]; }
    const sp = (c[2] || "").trim().split(/\s+/);
    const vp = (c[3] || "").trim().split(/\s+/);
    if (!name) continue;
    out.push({
      name, douyin,
      sales: (sp[0] || "").trim(),
      salesVolume: (vp[0] || "").trim(),
      aov: (c[4] || "").trim(),
      fans: (c[5] || "").trim(),
      category: (c[6] || "").trim(),
      liveSessions: (c[7] || "").trim(),
    });
  }
  return out;
}

async function currentPage(tab) {
  try { const a = tab.playwright.locator("li.number.active"); if (await a.count() > 0) { const n = parseInt((await a.first().innerText()).trim()); if (!isNaN(n)) return n; } } catch (e) {}
  return null;
}

async function clickNext(tab) {
  // Scroll to the bottom so the paginator is clickable
  try { await tab.cua.scroll({ x: 600, y: 400, scrollX: 0, scrollY: 5000 }); } catch (e) {}
  await tab.playwright.waitForTimeout(600);
  const cur = await currentPage(tab);
  // Primary: click the target page number (normal click)
  if (cur != null) {
    const target = cur + 1;
    try {
      const lis = tab.playwright.locator("li.number");
      const n = await lis.count();
      for (let i = 0; i < n; i++) {
        const t = (await lis.nth(i).innerText()).trim();
        if (t === String(target)) { await lis.nth(i).click(); await tab.playwright.waitForTimeout(2300); if (await currentPage(tab) === target) return true; break; }
      }
    } catch (e) {}
  }
  // Fallback: normal click on btn-next
  const btn = tab.playwright.locator(".btn-next");
  if (await btn.count() > 0) { try { await btn.click(); await tab.playwright.waitForTimeout(2300); if (cur == null || (await currentPage(tab)) !== cur) return true; } catch (e) {} }
  return false;
}

// Lower bound of a sales range in units of 10k ("12.5w~25w" -> 12.5)
function salesLowerBound(s) { if (!s) return 0; const m = s.match(/(\d+(?:\.\d+)?)\s*[w\u4e07]/); if (!m) return 0; let v = parseFloat(m[1]); return v; } // unit: 10k

function loadRank() { try { return JSON.parse(fs.readFileSync(RES, "utf8")); } catch (e) { return []; } }
function saveRank(arr) {
  // Dedupe by Douyin ID (unique), falling back to name
  const seen = new Set(); const out = [];
  for (const r of arr) { const k = r.douyin || r.name; if (k && !seen.has(k)) { seen.add(k); out.push(r); } }
  fs.writeFileSync(RES, JSON.stringify(out, null, 1));
}

// Scrape n pages starting from the current page, saving incrementally. Returns progress info.
export async function scrapeN(browser, n, category) {
  const tab = await getChanTab(browser);
  const rank = loadRank();
  const log = [];
  for (let p = 0; p < n; p++) {
    // Wait for table rows; retry / refresh when empty
    let rows = [];
    for (let attempt = 0; attempt < 3; attempt++) {
      rows = await extractPage(tab);
      if (rows.length > 0) break;
      await tab.playwright.waitForTimeout(2200);
      if (attempt === 1) {
        // Reload the ranking URL to recover
        try { await tab.goto("https://www.chanmama.com/bloggerRank/liveBlogger/?category_id=17&dateType=month&day=202608"); await tab.playwright.waitForLoadState({ state: "domcontentloaded" }); await tab.playwright.waitForTimeout(2500); } catch (e) {}
      }
    }
    if (rows.length === 0) { log.push({ page: "empty", stop: true }); break; }
    for (const r of rows) { rank.push({ ...r, category }); }
    saveRank(rank);
    const lastSales = rows[rows.length - 1].sales;
    const lb = salesLowerBound(lastSales);
    log.push({ extracted: rows.length, lastSales, lowerBound_w: lb });
    if (lb > 0 && lb < 10) { log[log.length - 1].stop = true; break; } // stop once the tail drops below 100k
    if (p < n - 1) {
      const ok = await clickNext(tab);
      if (!ok) { log.push({ page: "noNext", stop: true }); break; }
      await tab.playwright.waitForTimeout(1500);
    }
  }
  return { totalSaved: rank.length, log };
}

export async function gotoFoodRank(browser) {
  return gotoRank(browser, 17);
}
export async function gotoFreshRank(browser) {
  return gotoRank(browser, 3);
}
async function gotoRank(browser, catId) {
  const tab = await getChanTab(browser);
  await tab.goto(`https://www.chanmama.com/bloggerRank/liveBlogger/?category_id=${catId}&dateType=month&day=202608`);
  await tab.playwright.waitForLoadState({ state: "domcontentloaded" });
  await tab.playwright.waitForTimeout(2500);
  return await tab.url();
}

export function stats() {
  const rank = loadRank();
  return { total: rank.length, byCat: rank.reduce((a, r) => { a[r.category] = (a[r.category] || 0) + 1; return a; }, {}) };
}
