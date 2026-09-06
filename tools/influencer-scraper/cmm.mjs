// Chanmama influencer detail scraper module - driven from a browser-use JS kernel.
// Data directory: set the CMM_SCRAPER_DIR environment variable (default "./out").
// Expects an input file <dir>/top100.json (array of influencer display names) and writes
// incremental results to <dir>/chanmama_results.json.
// Note: string literals below are unicode-escaped Chinese UI labels of chanmama.com
// (a Chinese-language site); they are kept escaped so this source file stays ASCII-only.
import fs from "node:fs";

const DIR = (process.env.CMM_SCRAPER_DIR || "./out").replace(/\/+$/, "") + "/";

// Chinese UI strings of chanmama.com (unicode-escaped)
const LBL_SEARCH_PLACEHOLDER = "\u8fbe\u4eba\u540d\u79f0\u3001\u6296\u97f3\u53f7"; // "creator name / Douyin ID"
const LBL_FANS_TOTAL = "\u7c89\u4e1d\u603b\u6570";     // "total fans"
const LBL_SALES_LEVEL = "\u5e26\u8d27\u6c34\u5e73";    // "sales level"
const LBL_AVG_PRICE = "\u5e73\u5747\u5e26\u8d27\u4ef7"; // "average sales price"
const LBL_LIVE_SESSIONS = "\u76f4\u64ad\u573a\u6b21"; // "live sessions"
const LBL_CREATOR_ID = "\u8fbe\u4eba\u53f7";           // "creator ID"

export async function freshTab(browser) {
  const tabs = await browser.tabs.list();
  for (const t of tabs) { if (t.url && t.url.includes("chanmama")) { try { await (await browser.tabs.get(t.id)).close(); } catch (e) {} } }
  const tab = await browser.tabs.new();
  await tab.goto("https://www.chanmama.com/douyin/");
  await tab.playwright.waitForLoadState({ state: "domcontentloaded" });
  await tab.playwright.waitForTimeout(2000);
  return tab;
}

async function findInput(tab) {
  for (let a = 0; a < 3; a++) {
    const inp = tab.playwright.getByPlaceholder(LBL_SEARCH_PLACEHOLDER);
    if ((await inp.count()) > 0) {
      try { await inp.first().waitFor({ state: "visible" }); return inp.first(); } catch (e) {}
    }
    await tab.goto("https://www.chanmama.com/douyin/");
    await tab.playwright.waitForLoadState({ state: "domcontentloaded" });
    await tab.playwright.waitForTimeout(2000);
  }
  return null;
}

async function afterLabel(tab, label) {
  const loc = tab.playwright.getByText(label, { exact: true });
  if ((await loc.count()) === 0) return null;
  try { const s = loc.first().locator("xpath=following-sibling::*[1]"); if ((await s.count()) > 0) return (await s.first().textContent()).trim(); } catch (e) {}
  return null;
}

async function extractDetail(tab) {
  const F = {};
  F.fans = await afterLabel(tab, LBL_FANS_TOTAL);
  F.salesLevel = await afterLabel(tab, LBL_SALES_LEVEL);
  F.avgPrice = await afterLabel(tab, LBL_AVG_PRICE);
  F.liveSessions = await afterLabel(tab, LBL_LIVE_SESSIONS);
  // Sales ranges are rendered like "12.5w~25w" (w or the escaped wan char = 10k)
  const rng = tab.playwright.getByText(/^[\d][\d.]*[w\u4e07]~[\d][\d.]*[w\u4e07]$/);
  const ranges = await rng.allTextContents();
  F.sales = ranges[0] || null;
  F.salesVolume = ranges[1] || null;
  let cat = await afterLabel(tab, LBL_CREATOR_ID);
  if (cat) cat = cat.replace(/\d+(\.\d+)?%.*$/, '').replace(/\s+/g, '').trim();
  F.category = cat;
  return F;
}

function cleanQuery(raw) {
  // Full-width tilde/parens/colon (U+FF5E/U+FF08/U+FF09/U+FF1A) appear in raw Chinese names
  let q = raw.replace(/^[A-Za-z]\d*[~\uFF5E]\s*/, '').replace(/^B[~\uFF5E]\s*/, '').replace(/[\uFF08(].*[\uFF09)]/g, '').replace(/[~\uFF5E:.\uFF1A#]/g, '').trim();
  return q.split(/\s+/)[0];
}

// Parse a fan-count suffix such as "123.4w" (w or the escaped wan char = 10k)
function parseFans(t) {
  const re = new RegExp("\\u7c89\\u4e1d\\s*([\\d.,]+w?|[\\d.,]+\u4e07?)"); // "fans <count>"
  const m = t.match(re);
  return m ? m[1] : null;
}

function scoreCand(t, q) {
  const name = t.replace(new RegExp("\\u7c89\\u4e1d.*$"), '').replace(/[🍒🔥✨]/g, '').trim(); // strip "fans ..." suffix and emoji
  let score;
  if (name === q) score = 100;
  else if (name.startsWith(q)) score = 85;
  else if (name.includes(q)) score = 70;
  else if (q.includes(name) && name.length >= 2) score = 65;
  else { const ov = q.split('').filter(ch => name.includes(ch)).length; score = Math.round(ov / Math.max(q.length, 1) * 45); }
  return { score, name, fans: parseFans(t) };
}

// Food & beverage categories as listed on chanmama.com (unicode-escaped)
const FOOD = new RegExp("\u98df\u54c1|\u7f8e\u98df|\u751f\u9c9c|\u7cae\u6cb9|\u7c73\u9762|\u901f\u51bb|\u96f6\u98df|\u8c03\u5473|\u6c34\u4ea7|\u852c\u679c|\u5e72\u8d27|\u9884\u5236|\u70d8\u7119|\u8336|\u9152|\u6ecb\u8865");
const isFood = cat => !!(cat && FOOD.test(cat));
const salesGte10w = s => { if (!s) return false; const m = s.match(/(\d+(?:\.\d+)?)\s*[w\u4e07]/); if (!m) return false; return parseFloat(m[1]) >= 10; };
const liveCnt = s => { if (!s) return 0; const m = s.match(/(\d+)/); return m ? parseInt(m[1]) : 0; };

async function searchCandidates(tab, q) {
  const core = q.slice(0, Math.min(3, q.length));
  await tab.goto("https://www.chanmama.com/douyin/");
  await tab.playwright.waitForLoadState({ state: "domcontentloaded" });
  await tab.playwright.waitForTimeout(1500);
  const inp = await findInput(tab);
  if (!inp) return null;
  // Dropdown retry: 1st attempt fill; 2nd attempt type char-by-char (more likely to
  // trigger the suggestion dropdown)
  for (let attempt = 0; attempt < 2; attempt++) {
    try { await inp.click(); } catch (e) { await tab.playwright.waitForTimeout(800); await inp.click().catch(() => {}); }
    await inp.fill("");
    if (attempt === 0) { await inp.fill(q); }
    else { await inp.type(q, { delay: 60 }); }
    await tab.playwright.waitForTimeout(attempt === 0 ? 2300 : 2900);
    const links = tab.playwright.getByRole("link", { name: new RegExp(core) });
    const cnt = await links.count();
    const scored = [];
    for (let i = 0; i < Math.min(cnt, 10); i++) {
      const l = links.nth(i); let t = null, h = null;
      try { t = (await l.textContent()).trim().slice(0, 30); } catch (e) {}
      try { h = await l.getAttribute("href"); } catch (e) {}
      if (h && h.includes("bloggerRank") && t.includes(core)) scored.push({ ...scoreCand(t, q), h });
    }
    if (scored.length > 0) { scored.sort((a, b) => b.score - a.score); return scored; }
  }
  return [];
}

async function processOne(tab, raw) {
  const rec = { rawName: raw, queryName: null, matchedName: null, fans: null, verdict: null, fail: null };
  const q = cleanQuery(raw); rec.queryName = q;
  try {
    const scored = await searchCandidates(tab, q);
    if (scored === null) { rec.verdict = "error"; rec.fail = "search box not ready"; return rec; }
    if (scored.length === 0) { rec.verdict = "not_found"; return rec; }
    rec.candidates = scored.slice(0, 3).map(s => `${s.name}(${s.fans})`).join(" / ");
    let chosen = scored[0], F = null;
    for (let i = 0; i < Math.min(scored.length, 2); i++) {  // prefer food creators, inspect up to 2 candidates
      const c = scored[i];
      await tab.goto("https://www.chanmama.com" + c.h);
      await tab.playwright.waitForLoadState({ state: "domcontentloaded" });
      await tab.playwright.waitForTimeout(2300);
      const f = await extractDetail(tab);
      if (i === 0) { chosen = c; F = f; }
      if (isFood(f.category)) { chosen = c; F = f; break; }
    }
    rec.matchedName = chosen.name;
    rec.fans = chosen.fans;
    rec.confidence = chosen.score >= 85 ? "high" : (chosen.score >= 65 ? "mid" : "low");
    Object.assign(rec, F);
    const food = isFood(rec.category), salesOK = salesGte10w(rec.sales), liveOK = liveCnt(rec.liveSessions) > 0;
    const fails = [];
    if (!food) fails.push("not-food");
    if (!salesOK) fails.push("sales<10w");
    if (!liveOK) fails.push("no-livestream");
    rec.verdict = (food && salesOK && liveOK) ? "target" : "not_target";
    rec.fail = fails.length ? fails.join("+") : null;
  } catch (e) { rec.verdict = "error"; rec.fail = (e.message || String(e)).slice(0, 60); }
  return rec;
}

export async function runBatch(browser, START, BATCH) {
  const TOP = JSON.parse(await fs.promises.readFile(DIR + "top100.json", "utf8"));
  const RES_PATH = DIR + "chanmama_results.json";
  let results = {}; try { results = JSON.parse(await fs.promises.readFile(RES_PATH, "utf8")); } catch (e) {}
  const slice = TOP.slice(START, START + BATCH);
  const tab = await freshTab(browser);
  const out = [];
  for (const n of slice) {
    const r = await processOne(tab, n);
    results[n] = r; out.push(r);
    await fs.promises.writeFile(RES_PATH, JSON.stringify(results, null, 2));
    await tab.playwright.waitForTimeout(900);
  }
  return { range: [START + 1, Math.min(START + BATCH, TOP.length)], done: out.length, verdicts: out };
}

export async function retryFailed(browser) {
  const TOP = JSON.parse(await fs.promises.readFile(DIR + "top100.json", "utf8"));
  const RES_PATH = DIR + "chanmama_results.json";
  let results = {}; try { results = JSON.parse(await fs.promises.readFile(RES_PATH, "utf8")); } catch (e) {}
  const todo = TOP.filter(n => !results[n] || results[n].verdict === "not_found" || results[n].verdict === "error");
  if (todo.length === 0) return { retried: 0, left: 0 };
  const tab = await freshTab(browser);
  const out = [];
  for (const n of todo) {
    const r = await processOne(tab, n);
    results[n] = r;
    out.push({ name: n, verdict: r.verdict, matched: r.matchedName, fans: r.fans, fail: r.fail });
    await fs.promises.writeFile(RES_PATH, JSON.stringify(results, null, 2));
    await tab.playwright.waitForTimeout(1000);
  }
  const stillLeft = TOP.filter(n => !results[n] || results[n].verdict === "not_found" || results[n].verdict === "error").length;
  return { retried: todo.length, left: stillLeft, results: out };
}

export async function status() {
  const TOP = JSON.parse(await fs.promises.readFile(DIR + "top100.json", "utf8"));
  const RES_PATH = DIR + "chanmama_results.json";
  let results = {}; try { results = JSON.parse(await fs.promises.readFile(RES_PATH, "utf8")); } catch (e) {}
  const tally = {};
  let done = 0, targets = [];
  for (const n of TOP) {
    const r = results[n];
    if (!r) { tally["pending"] = (tally["pending"] || 0) + 1; continue; }
    done++;
    tally[r.verdict] = (tally[r.verdict] || 0) + 1;
    if (r.verdict === "target") targets.push({ name: n, matched: r.matchedName, fans: r.fans, category: r.category, sales: r.sales, lives: r.liveSessions });
  }
  return { done, tally, targets };
}
