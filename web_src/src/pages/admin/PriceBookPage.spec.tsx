import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";

import PriceBookPage from "./PriceBookPage";

const book = {
  version: "2026-08-31.2",
  effective_at: "2026-08-31T12:00:00Z",
  model_rate_count: 1,
  compute_rate_count: 1,
  rates: [
    {
      usage_kind: "model",
      match_key: "claude-sonnet",
      match_mode: "prefix",
      input_cents_per_million: 300,
      output_cents_per_million: 1500,
      cache_read_cents_per_million: 30,
      cache_write_cents_per_million: 375,
      reasoning_cents_per_million: 0,
      micros_per_second: 0,
    },
    {
      usage_kind: "compute",
      match_key: "e1-large-amd64",
      match_mode: "exact",
      input_cents_per_million: 0,
      output_cents_per_million: 0,
      cache_read_cents_per_million: 0,
      cache_write_cents_per_million: 0,
      reasoning_cents_per_million: 0,
      micros_per_second: 556,
    },
  ],
};

const mockFetch = (handler: (url: string, init?: RequestInit) => Promise<Response> | Response) => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      return handler(String(input), init);
    }),
  );
};

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/admin/price-book"]}>
      <PriceBookPage />
    </MemoryRouter>,
  );

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("PriceBookPage", () => {
  it("loads the current catalog and filters rates", async () => {
    mockFetch(async () => {
      return new Response(JSON.stringify(book), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    const user = userEvent.setup();

    renderPage();

    expect(await screen.findByTestId("admin-price-book-version")).toHaveTextContent("2026-08-31.2");
    expect(screen.getByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByText("e1-large-amd64")).toBeInTheDocument();

    await user.type(screen.getByTestId("admin-price-book-search"), "e1-large");
    expect(screen.queryByText("claude-sonnet")).not.toBeInTheDocument();
    expect(screen.getByText("e1-large-amd64")).toBeInTheDocument();
  });

  it("scans and updates the catalog", async () => {
    mockFetch(async (url, init) => {
      if (url.endsWith("/scan") && init?.method === "POST") {
        return new Response(
          JSON.stringify({
            ...book,
            version: "2026-09-09.1",
            previous_version: "2026-08-31.2",
            model_rate_count: 2,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response(JSON.stringify(book), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    const user = userEvent.setup();

    renderPage();
    expect(await screen.findByTestId("admin-price-book-version")).toHaveTextContent("2026-08-31.2");

    await user.click(screen.getByTestId("admin-price-book-scan"));
    await waitFor(() => {
      expect(screen.getByTestId("admin-price-book-version")).toHaveTextContent("2026-09-09.1");
    });
  });
});
