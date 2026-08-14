# amsterdam-ratings

Estimates a Google Maps rating **normalization matrix** for Amsterdam
neighborhoods: how many stars to add/subtract per area, holding category
mix and review volume constant, so a 4.6 in a touristy area and a 4.2 in a
residential one can be compared fairly.

Methodology writeup / background: cross-platform rating inflation research
(Raval 2024, "Do Bad Businesses Get Good Reviews?") shows Google ratings
inflate more for low-quality businesses than honest platforms do, and that
short/no-text reviews (which Google allows and Yelp doesn't) are a
measurable driver. This project transfers that *mechanism* — review effort
as an inflation signal — to a within-Google, cross-neighborhood setting,
since there's no Amsterdam-specific "ground truth" platform to compare
against directly.

## Pipeline

```
cmd/fetch       -> Places API sweep (Nearby Search + optional Place Details)     -> data/places.json
cmd/fetch-yelp  -> Yelp Fusion API sweep, same neighborhoods/categories          -> data/yelp.json
(tripadvisor)   -> manual/partner CSV export (no public API - see below)         -> data/tripadvisor.csv
cmd/analyze     -> OLS regression (neighborhood + category dummies + log(review_count)) -> within-Google normalization matrix
cmd/compare     -> cross-source venue matching + per-neighborhood rating gap     -> Google vs. Yelp vs. TripAdvisor deltas
```

`cmd/analyze` answers "how do Amsterdam neighborhoods compare to each other
*on Google*." `cmd/compare` answers a different question: "how much higher
does Google rate the *same* venues than Yelp/TripAdvisor do, and does that
gap vary by neighborhood" — the direct cross-platform test the README below
used to only suggest as a future extension.

No third-party Go dependencies — everything (including the OLS solver and
the cross-source name/distance matcher) is stdlib only, so it builds with
just `go build ./...`, no `go mod tidy` required, no module proxy needed.

## Try it without an API key

```bash
go run ./cmd/fetch -dry-run -out data/places.json -limit 15
go run ./cmd/analyze -in data/places.json
```

This generates synthetic data with a *known* injected neighborhood bias and
review-effort pattern (Centrum/Jordaan inflated + low-effort reviews,
Noord/Zuidoost deflated + longer reviews) so you can confirm the regression
recovers something sane before spending real API quota.

The cross-platform comparison has its own dry-run, which also synthesizes
Yelp and TripAdvisor data (each with its own smaller injected bias) and
matches them against the synthetic Google data by name + distance:

```bash
go run ./cmd/compare -dry-run
```

## Run it for real

Requires a Google Cloud project with the **Places API** enabled and billing
configured. Nearby Search and Place Details are both billed per call — see
[Google's Places API pricing](https://developers.google.com/maps/documentation/places/web-service/usage-and-billing)
before running at scale. With 10 neighborhoods x 5 categories x ~20 places
x (1 details call each if `-with-reviews`), expect on the order of a
thousand+ billed calls — check current pricing and set a budget cap in
Cloud Console first.

```bash
export GOOGLE_MAPS_API_KEY=your_key_here

# Ratings + review counts only (cheaper, no review text -> no effort diagnostics)
go run ./cmd/fetch -out data/places.json -limit 20

# Also pull up to 5 sample reviews per place (adds 1 billed Place Details call per place)
go run ./cmd/fetch -out data/places.json -limit 20 -with-reviews

go run ./cmd/analyze -in data/places.json
```

### Cross-platform comparison (Google vs. Yelp vs. TripAdvisor)

Yelp Fusion has a free-tier API — sign up at
https://docs.developer.yelp.com/docs/fusion-authentication for a key:

```bash
export YELP_API_KEY=your_key_here
go run ./cmd/fetch-yelp -out data/yelp.json -limit 20
```

TripAdvisor has **no self-serve public API** as of 2026 — its Content API
is partner-gated. `internal/sources/tripadvisor` has no live-fetch client
because there's nothing to build against; instead it loads a CSV you
compile by hand or via an approved partner/licensed export, with header
`name,neighborhood,category,lat,lng,rating,review_count,external_id`.
TripAdvisor is optional — `cmd/compare` runs fine with just Google + Yelp.

```bash
go run ./cmd/compare -google data/places.json -yelp data/yelp.json -tripadvisor data/tripadvisor.csv
```

This matches venues across sources (nearest by distance, filtered by
name-token similarity — see `internal/match`) and reports, per neighborhood,
the average rating gap between Google and each other platform plus match
coverage (what fraction of Google venues found a match — low coverage means
treat that neighborhood's gap as low-confidence). Tune matching strictness
with `-max-distance-m` (default 120) and `-min-name-score` (default 0.3).

## Extending

- **Neighborhoods/categories**: edit `internal/config/config.go`. Current
  centroids are approximate — tighten radii or split Centrum into
  sub-areas (e.g. De Wallen vs. rest) if you want finer granularity.
- **More covariates**: `internal/features/BuildDesignMatrix` is where to
  add e.g. a foreign-language-review share, price level, or distance to
  Dam Square as additional controls.
- **Matching precision**: `internal/match` uses name-token Jaccard
  similarity + a distance cutoff, which is a reasonable heuristic but will
  miss venues with very different names across platforms (e.g. a
  translated/rebranded listing) and can mismatch generic names (e.g. two
  different "Café Central"s within `-max-distance-m` of each other). If
  match coverage is consistently low in some neighborhood, check that
  before trusting its cross-platform gap.
- **Shrinkage**: for a per-venue "adjusted rating" (not just the aggregate
  matrix), add Bayesian shrinkage toward the category+neighborhood mean,
  weighted by review count, before subtracting the neighborhood
  coefficient — low-review-count places are the noisiest and most likely
  to be a false positive for "inflated."

## Caveats

- Nearby Search + Place Details reflect Google's current, live index —
  re-running later will give different numbers as reviews accrue.
- Place Details only returns ~5 "most relevant" reviews per place, so the
  review-effort diagnostics are a noisy per-neighborhood aggregate, not a
  precise per-venue measure — that's why they're reported as a diagnostic
  alongside the regression, not fed directly into it as a per-place
  covariate.
- This estimates *relative* neighborhood bias, not absolute truth — there's
  no independent quality ground truth (no Amsterdam BBB-equivalent), so
  treat the matrix as "how areas compare to each other on Google," not
  "how much each area's ratings deviate from real quality."
