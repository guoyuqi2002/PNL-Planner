# influencer-scraper

Two Node.js browser-automation modules used to collect influencer analytics from
chanmama.com (a public Douyin data platform), written while researching candidate
influencers for live-commerce campaigns.

- `rankscraper.mjs` - walks the live-selling influencer ranking (paginated table),
  extracting name, Douyin ID, sales range, sales volume, AOV, fan count, category and
  live-session count per row; stops once the tail of the ranking drops below a sales
  threshold. Results accumulate in `ranking.json`.
- `cmm.mjs` - given a list of influencer names (`top100.json`), searches each one on
  chanmama.com, opens the best match, reads profile stats (fans, sales level, average
  price, live sessions, main category) and classifies it as target / not-target based on
  category (food & beverage), sales level and livestream activity. Results accumulate in
  `chanmama_results.json`.

## Usage

```bash
export CMM_SCRAPER_DIR=./out   # data dir (default ./out); expects top100.json inside
node --input-type=module -e "import('./cmm.mjs').then(m => m.status())"
```

Both modules export async functions (`runBatch`, `retryFailed`, `status`, `scrapeN`,
`gotoFoodRank`, ...) intended to be driven by a browser-automation runtime that provides
a `browser` object with Playwright-compatible tab APIs.

## Disclaimer

These scripts were used for authorized market research during an internship. They respect
the target site's public pages only and are shared for portfolio purposes. Chinese UI
selectors of chanmama.com are kept as unicode escapes; adjust them if the site changes.
