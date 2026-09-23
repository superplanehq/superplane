import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

import {
  currentCatalog,
  jsonResponse,
  middleCatalog,
  olderCatalog,
  renderPage,
  savedCatalog,
} from "./priceBooksTestSupport";

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

afterEach(() => {
  vi.unstubAllGlobals();
  vi.mocked(showErrorToast).mockClear();
  vi.mocked(showSuccessToast).mockClear();
});

describe("PriceBooks catalog", () => {
  it("loads the current catalog", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(currentCatalog)),
    );

    renderPage();

    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-version")).toHaveTextContent("2026-09-09.1 (current)");
    expect(screen.getByDisplayValue("3.00")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-sync")).toBeInTheDocument();
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

    await user.click(screen.getByTestId("admin-price-book-version"));
    await user.click(await screen.findByRole("option", { name: "2026-08-31.1" }));
    await user.click(screen.getByTestId("admin-price-book-version"));
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
    expect(screen.getByTestId("admin-price-book-activate")).toBeInTheDocument();
    expect(screen.queryByTestId("admin-price-book-sync")).not.toBeInTheDocument();
  });

  it("saves edited rates and adds a VM row", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT" && String(input) === "/admin/api/price-books") {
        const body = JSON.parse(String(init.body)) as {
          base_version: string;
          models: { input_cents_per_million: number }[];
          vms: { match_key: string }[];
        };
        expect(body.base_version).toBe("2026-09-09.1");
        expect(body.models[0].input_cents_per_million).toBe(400);
        expect(body.vms.some((rate) => rate.match_key === "e1-test-amd64")).toBe(true);
        return jsonResponse(savedCatalog);
      }
      return jsonResponse(currentCatalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    const input = screen.getByDisplayValue("3.00");
    await user.clear(input);
    await user.type(input, "4");
    await user.tab();

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    await user.type(screen.getByLabelText("Machine type", { selector: "#price-book-add-vm-key" }), "e1-test-amd64");
    await user.click(screen.getByRole("button", { name: "Add machine rate" }));
    await user.click(screen.getByTestId("admin-price-book-save-vms"));

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
    expect(await screen.findByTestId("admin-price-book-version")).toHaveTextContent("2026-09-13.1 (current)");
  });

  it("updates model rates from the provider", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "POST" && String(input).startsWith("/admin/api/price-books/sync")) {
          return jsonResponse({ ...savedCatalog, updated_count: 1, added_count: 0, skipped_providers: [] });
        }
        return jsonResponse(currentCatalog);
      }),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-price-book-sync"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
    expect(screen.getByTestId("admin-price-book-version")).toHaveTextContent("2026-09-13.1 (current)");
  });

  it("activates an older version", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "PUT" && String(input) === "/admin/api/price-books/current") {
          return jsonResponse({ ...olderCatalog, current_version: "2026-08-31.1" });
        }
        const url = String(input);
        if (url.includes("version=2026-08-31.1")) {
          return jsonResponse(olderCatalog);
        }
        return jsonResponse(currentCatalog);
      }),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-price-book-version"));
    await user.click(await screen.findByRole("option", { name: "2026-08-31.1" }));
    expect(await screen.findByText("older-model")).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-price-book-activate"));
    await user.click(screen.getByTestId("admin-price-book-activate-confirm"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
    expect(screen.getByTestId("admin-price-book-version")).toHaveTextContent("2026-08-31.1 (current)");
    expect(screen.getByTestId("admin-price-book-sync")).toBeInTheDocument();
    expect(screen.queryByTestId("admin-price-book-activate")).not.toBeInTheDocument();
    expect(screen.getByDisplayValue("3.00")).toBeInTheDocument();
  });

  it("disables save while a version change is loading", async () => {
    const olderRequest = new Promise<Response>(() => undefined);

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("version=2026-08-31.1")) {
          return olderRequest;
        }
        return jsonResponse(currentCatalog);
      }),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByTestId("admin-price-book-version"));
    await user.click(await screen.findByRole("option", { name: "2026-08-31.1" }));

    await waitFor(() => {
      expect(screen.getByTestId("admin-price-book-save")).toBeDisabled();
    });
    expect(screen.getByTestId("admin-price-book-sync")).toBeDisabled();
  });

  it("disables activate until the selected version finishes loading", async () => {
    let resolveMiddle: ((response: Response) => void) | undefined;
    const middleRequest = new Promise<Response>((resolve) => {
      resolveMiddle = resolve;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("version=2026-09-01.1")) {
          return middleRequest;
        }
        if (url.includes("version=2026-08-31.1")) {
          return jsonResponse(olderCatalog);
        }
        return jsonResponse(currentCatalog);
      }),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-price-book-version"));
    await user.click(await screen.findByRole("option", { name: "2026-08-31.1" }));
    expect(await screen.findByText("older-model")).toBeInTheDocument();

    await user.click(screen.getByTestId("admin-price-book-version"));
    await user.click(await screen.findByRole("option", { name: "2026-09-01.1" }));

    await waitFor(() => {
      expect(screen.getByTestId("admin-price-book-activate")).toBeDisabled();
    });

    await act(async () => {
      resolveMiddle?.(jsonResponse(middleCatalog));
      await middleRequest;
    });

    expect(await screen.findByText("middle-model")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-activate")).toBeEnabled();
  });
});
