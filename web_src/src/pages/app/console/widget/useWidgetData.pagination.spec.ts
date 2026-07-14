import { describe, it, expect } from "vitest";

import {
  DEFAULT_AGGREGATE_LIMIT,
  INITIAL_EAGER_ROWS,
  WIDGET_MAX_EAGER_PAGES,
  computeDisplaySlice,
  computeEffectiveLimit,
  computeInitialDisplayCount,
  computeTrendCollectLimit,
  computeWidgetHasMore,
  isWidgetQueryLoading,
  shouldFetchNextWidgetPage,
  splitDisplayRowsWithTrendPeek,
} from "./useWidgetData";

describe("computeEffectiveLimit", () => {
  it("returns the raw limit when it is a positive finite number", () => {
    expect(computeEffectiveLimit(50, false)).toBe(50);
    expect(computeEffectiveLimit(250, true)).toBe(250);
  });

  it("falls back to Infinity for progressive callers with a blank limit", () => {
    expect(computeEffectiveLimit(undefined, true)).toBe(Number.POSITIVE_INFINITY);
    expect(computeEffectiveLimit(0, true)).toBe(Number.POSITIVE_INFINITY);
  });

  it("falls back to DEFAULT_AGGREGATE_LIMIT for non-progressive callers with a blank limit", () => {
    expect(computeEffectiveLimit(undefined, false)).toBe(DEFAULT_AGGREGATE_LIMIT);
    expect(computeEffectiveLimit(-5, false)).toBe(DEFAULT_AGGREGATE_LIMIT);
  });
});

describe("computeInitialDisplayCount", () => {
  it("returns the full effective limit for non-progressive callers", () => {
    expect(computeInitialDisplayCount(50, false)).toBe(50);
    expect(computeInitialDisplayCount(500, false)).toBe(500);
  });

  it("returns INITIAL_EAGER_ROWS for progressive callers with an unbounded limit", () => {
    expect(computeInitialDisplayCount(Number.POSITIVE_INFINITY, true)).toBe(INITIAL_EAGER_ROWS);
  });

  it("clamps the initial window to a finite effective limit", () => {
    expect(computeInitialDisplayCount(30, true)).toBe(30);
    expect(computeInitialDisplayCount(INITIAL_EAGER_ROWS * 2, true)).toBe(INITIAL_EAGER_ROWS);
  });
});

describe("computeDisplaySlice", () => {
  it("clamps display count to the effective limit", () => {
    expect(computeDisplaySlice(150, 100)).toBe(100);
    expect(computeDisplaySlice(50, 100)).toBe(50);
    expect(computeDisplaySlice(50, Number.POSITIVE_INFINITY)).toBe(50);
  });
});

describe("computeTrendCollectLimit", () => {
  it("collects every already-loaded row so filter+sort can see hidden baselines", () => {
    expect(computeTrendCollectLimit(100, 125)).toBe(125);
  });

  it("matches the display window when nothing is hidden", () => {
    expect(computeTrendCollectLimit(100, 100)).toBe(100);
    expect(computeTrendCollectLimit(100, 80)).toBe(100);
  });

  it("caps at the configured effective limit", () => {
    expect(computeTrendCollectLimit(50, 200, 120)).toBe(120);
  });
});

describe("splitDisplayRowsWithTrendPeek", () => {
  it("returns the full list with no peek when nothing is hidden", () => {
    const rows = [{ id: 1 }, { id: 2 }];
    expect(splitDisplayRowsWithTrendPeek(rows, 10)).toEqual({ rows, nextLoadedRow: undefined });
    expect(splitDisplayRowsWithTrendPeek(rows, 2)).toEqual({ rows, nextLoadedRow: undefined });
  });

  it("splits the first hidden row out as nextLoadedRow", () => {
    const rows = [{ id: 1 }, { id: 2 }, { id: 3 }];
    expect(splitDisplayRowsWithTrendPeek(rows, 2)).toEqual({
      rows: [{ id: 1 }, { id: 2 }],
      nextLoadedRow: { id: 3 },
    });
  });
});

describe("computeWidgetHasMore", () => {
  const baseInput = {
    progressive: true,
    displayCount: 100,
    effectiveLimit: Number.POSITIVE_INFINITY,
    loadedRowCount: 100,
    hasNextPage: false as boolean | undefined,
    pageCount: 4,
  };

  it("returns false for non-progressive callers", () => {
    expect(computeWidgetHasMore({ ...baseInput, progressive: false, hasNextPage: true })).toBe(false);
  });

  it("returns false when displayCount has caught up to the effective limit", () => {
    expect(computeWidgetHasMore({ ...baseInput, displayCount: 100, effectiveLimit: 100, hasNextPage: true })).toBe(
      false,
    );
  });

  it("returns true when more loaded rows can be revealed without fetching", () => {
    expect(computeWidgetHasMore({ ...baseInput, loadedRowCount: 200, hasNextPage: false })).toBe(true);
  });

  it("returns true when more pages can still be fetched within the page budget", () => {
    expect(computeWidgetHasMore({ ...baseInput, hasNextPage: true, pageCount: 4 })).toBe(true);
  });

  it("returns false when the eager-page budget is exhausted and no extra rows are loaded", () => {
    expect(computeWidgetHasMore({ ...baseInput, hasNextPage: true, pageCount: WIDGET_MAX_EAGER_PAGES })).toBe(false);
  });

  it("treats hasNextPage===undefined the same as false", () => {
    expect(computeWidgetHasMore({ ...baseInput, hasNextPage: undefined })).toBe(false);
  });

  it("returns false when callers suppress hasNextPage for unmatchable trigger filters", () => {
    // useRunsDataSourceResult passes hasNextPage: false when triggersMatchable
    // is false so Load more is hidden for stale trigger YAML.
    expect(computeWidgetHasMore({ ...baseInput, hasNextPage: false, loadedRowCount: 0 })).toBe(false);
  });
});

describe("shouldFetchNextWidgetPage", () => {
  const baseInput = {
    enabled: true,
    fillTarget: 100,
    loadedRowCount: 25,
    pageCount: 1,
    hasNextPage: true as boolean | undefined,
    isFetchingNextPage: false,
    isFetching: false,
  };

  it("fetches when below the fill target with budget and no in-flight request", () => {
    expect(shouldFetchNextWidgetPage(baseInput)).toBe(true);
  });

  it("does not fetch when disabled", () => {
    expect(shouldFetchNextWidgetPage({ ...baseInput, enabled: false })).toBe(false);
  });

  it("does not fetch when there is no next page", () => {
    expect(shouldFetchNextWidgetPage({ ...baseInput, hasNextPage: false })).toBe(false);
  });

  it("does not fetch while a fetch is already in flight", () => {
    expect(shouldFetchNextWidgetPage({ ...baseInput, isFetchingNextPage: true })).toBe(false);
    expect(shouldFetchNextWidgetPage({ ...baseInput, isFetching: true })).toBe(false);
  });

  it("does not fetch when the fill target is already satisfied", () => {
    expect(shouldFetchNextWidgetPage({ ...baseInput, loadedRowCount: 100 })).toBe(false);
  });

  it("does not fetch beyond the page budget", () => {
    expect(shouldFetchNextWidgetPage({ ...baseInput, pageCount: WIDGET_MAX_EAGER_PAGES })).toBe(false);
  });
});

describe("isWidgetQueryLoading", () => {
  const baseInput = {
    queryIsLoading: false,
    enabled: true,
    hasNextPage: true as boolean | undefined,
    loadedRowCount: 25,
    fillTarget: 100,
    pageCount: 1,
    isFetchingNextPage: true,
    isFetching: true,
  };

  it("reports loading while the initial fill is still in progress", () => {
    expect(isWidgetQueryLoading(baseInput)).toBe(true);
  });

  it("reports loading when the underlying query reports its initial fetch", () => {
    expect(isWidgetQueryLoading({ ...baseInput, queryIsLoading: true, isFetchingNextPage: false })).toBe(true);
  });

  it("stops reporting loading once the fill target is satisfied", () => {
    expect(isWidgetQueryLoading({ ...baseInput, loadedRowCount: 100 })).toBe(false);
  });

  it("stops reporting loading when there are no more pages", () => {
    expect(isWidgetQueryLoading({ ...baseInput, hasNextPage: false })).toBe(false);
  });

  it("respects the eager-page budget", () => {
    expect(isWidgetQueryLoading({ ...baseInput, pageCount: WIDGET_MAX_EAGER_PAGES })).toBe(false);
  });
});
