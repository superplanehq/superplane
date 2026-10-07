import { describe, expect, it } from "bun:test";

import { shouldLoadNextBacklogSearchPage } from "./backlogColumnSearch";

const readyToLoad = {
  query: "refund",
  hasMore: true,
  isLoadingNextPage: false,
  hasPageError: false,
};

describe("shouldLoadNextBacklogSearchPage", () => {
  it("loads the next page while a query still has pages to fetch", () => {
    expect(shouldLoadNextBacklogSearchPage(readyToLoad)).toBe(true);
    expect(shouldLoadNextBacklogSearchPage({ ...readyToLoad, query: "  refund  " })).toBe(true);
  });

  it.each([
    { ...readyToLoad, query: "" },
    { ...readyToLoad, query: "   " },
    { ...readyToLoad, hasMore: false },
    { ...readyToLoad, isLoadingNextPage: true },
    { ...readyToLoad, hasPageError: true },
  ])("does not load when the search cannot fetch another page (%j)", (args) => {
    expect(shouldLoadNextBacklogSearchPage(args)).toBe(false);
  });
});
