import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { showErrorToast } from "@/lib/toast";

import { PriceBooks } from "./PriceBooks";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

const versions = [
  { version: "2026-09-09.1", effective_at: "2026-09-09T00:00:00Z", created_at: "2026-09-09T00:00:00Z" },
  { version: "2026-09-01.1", effective_at: "2026-09-01T00:00:00Z", created_at: "2026-09-01T00:00:00Z" },
  { version: "2026-08-31.1", effective_at: "2026-08-31T00:00:00Z", created_at: "2026-08-31T00:00:00Z" },
];

const catalogFor = (version: string, matchKey: string) => ({
  current_version: "2026-09-09.1",
  version,
  effective_at: `${version.slice(0, 10)}T00:00:00Z`,
  created_at: `${version.slice(0, 10)}T00:00:00Z`,
  versions,
  models: [
    {
      match_key: matchKey,
      match_mode: "prefix",
      input_cents_per_million: 300,
      output_cents_per_million: 1500,
      cache_read_cents_per_million: 30,
      cache_write_cents_per_million: 375,
      reasoning_cents_per_million: 0,
    },
  ],
  vms: [
    {
      match_key: "e1-large-amd64",
      match_mode: "exact",
      micros_per_second: 70,
    },
  ],
});

const currentCatalog = catalogFor("2026-09-09.1", "claude-sonnet");
const olderCatalog = catalogFor("2026-08-31.1", "older-model");
const middleCatalog = catalogFor("2026-09-01.1", "middle-model");

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/admin/price-books"]}>
      <PriceBooks />
    </MemoryRouter>,
  );

afterEach(() => {
  vi.unstubAllGlobals();
  vi.mocked(showErrorToast).mockClear();
});

describe("PriceBooks", () => {
  it("loads the current catalog", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(currentCatalog)),
    );

    renderPage();

    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-version")).toHaveTextContent("2026-09-09.1 (current)");
    expect(screen.getByText("$3.00")).toBeInTheDocument();
  });

  it("keeps the latest selected version when an earlier request finishes last", async () => {
    let resolveOlder: ((response: Response) => void) | undefined;
    const olderRequest = new Promise<Response>((resolve) => {
      resolveOlder = resolve;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("version=2026-08-31.1")) {
          return olderRequest;
        }
        if (url.includes("version=2026-09-01.1")) {
          return jsonResponse(middleCatalog);
        }
        return jsonResponse(currentCatalog);
      }),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "2026-08-31.1" }));
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "2026-09-01.1" }));

    expect(await screen.findByText("middle-model")).toBeInTheDocument();

    await act(async () => {
      resolveOlder?.(jsonResponse(olderCatalog));
      await olderRequest;
    });

    await waitFor(() => {
      expect(screen.getByText("middle-model")).toBeInTheDocument();
    });
    expect(screen.queryByText("older-model")).not.toBeInTheDocument();
    expect(screen.queryByText("claude-sonnet")).not.toBeInTheDocument();
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});
