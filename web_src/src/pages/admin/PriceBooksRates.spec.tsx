import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { currentCatalog, jsonResponse, olderCatalog, renderPage, savedCatalog } from "./priceBooksTestSupport";

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

describe("PriceBooks machine rates", () => {
  it("removes a machine rate and saves the remaining row", async () => {
    const catalog = {
      ...currentCatalog,
      vms: [
        { match_key: "e1-large-amd64", match_mode: "exact", micros_per_second: 70 },
        { match_key: "e1-small-amd64", match_mode: "exact", micros_per_second: 40 },
      ],
    };
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT" && String(input) === "/admin/api/price-books") {
        const body = JSON.parse(String(init.body)) as {
          vms: { match_key: string; micros_per_second: number }[];
        };
        expect(body.vms).toEqual([{ match_key: "e1-large-amd64", match_mode: "exact", micros_per_second: 70 }]);
        return jsonResponse(savedCatalog);
      }
      return jsonResponse(catalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    expect(screen.getByText("e1-large-amd64")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("e1-large-amd64")).not.toBeInTheDocument();

    const removeButtons = screen.getAllByRole("button", { name: "Remove" });
    await user.click(removeButtons[1]);
    expect(screen.queryByText("e1-small-amd64")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("admin-price-book-save-vms"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
  });

  it("rejects empty and duplicate VM machine types", async () => {
    const catalog = {
      ...currentCatalog,
      vms: [
        { match_key: "e1-large-amd64", match_mode: "exact", micros_per_second: 70 },
        { match_key: "e1-small-amd64", match_mode: "exact", micros_per_second: 40 },
      ],
    };
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT" && String(input) === "/admin/api/price-books") {
        const body = JSON.parse(String(init.body)) as {
          vms: { match_key: string; micros_per_second: number }[];
        };
        expect(body.vms).toEqual([
          { match_key: "e1-large-amd64", match_mode: "exact", micros_per_second: 70 },
          { match_key: "e1-small-amd64", match_mode: "exact", micros_per_second: 40 },
        ]);
        return jsonResponse(savedCatalog);
      }
      return jsonResponse(catalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    const machineType = screen.getByLabelText("Machine type", { selector: "#price-book-add-vm-key" });

    await user.click(screen.getByRole("button", { name: "Add machine rate" }));
    expect(showErrorToast).toHaveBeenCalledWith("Enter a machine type.");

    await user.type(machineType, "   ");
    await user.click(screen.getByRole("button", { name: "Add machine rate" }));
    expect(showErrorToast).toHaveBeenCalledWith("Enter a machine type.");

    await user.clear(machineType);
    await user.type(machineType, "e1-small-amd64");
    await user.click(screen.getByRole("button", { name: "Add machine rate" }));
    expect(showErrorToast).toHaveBeenCalledWith("That machine rate already exists.");

    await user.click(screen.getByTestId("admin-price-book-save-vms"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
  });

  it("saves an empty VM list after the last row is removed", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT" && String(input) === "/admin/api/price-books") {
        const body = JSON.parse(String(init.body)) as { vms: unknown[] };
        expect(body.vms).toEqual([]);
        return jsonResponse({ ...savedCatalog, vms: [] });
      }
      return jsonResponse(currentCatalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(screen.getByText("This version has no machine rates.")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-save-vms")).toBeEnabled();

    await user.click(screen.getByTestId("admin-price-book-save-vms"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
  });

  it("keeps older VM rows read-only", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
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

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    expect(screen.getByText("e1-large-amd64")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Machine type" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  });
});

describe("PriceBooks unused models", () => {
  it("shows selected models and keeps unused models collapsed", async () => {
    const catalog = {
      ...currentCatalog,
      models: [
        {
          ...currentCatalog.models[0],
          selected: true,
        },
        {
          provider: "openrouter",
          match_key: "other-model",
          match_mode: "exact",
          input_cents_per_million: 100,
          output_cents_per_million: 500,
          cache_read_cents_per_million: 10,
          cache_write_cents_per_million: 50,
          reasoning_cents_per_million: 0,
          selected: false,
        },
      ],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(catalog)),
    );

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Selected models")).toBeInTheDocument();
    expect(screen.getByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-unused-toggle")).toHaveTextContent("Unused models (1)");
    expect(screen.queryByText("other-model")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("admin-price-book-unused-toggle"));
    expect(screen.getByText("other-model")).toBeInTheDocument();
  });

  it("shows empty state when no model is selected", async () => {
    const catalog = {
      ...currentCatalog,
      models: [{ ...currentCatalog.models[0], selected: false }],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(catalog)),
    );

    renderPage();

    expect(await screen.findByText("Selected models")).toBeInTheDocument();
    expect(screen.getByText("No models are selected in Hosted LLM settings.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open Hosted LLM settings" })).toHaveAttribute("href", "/admin/settings");
  });

  it("shows empty unused models after expand when every model is selected", async () => {
    const catalog = {
      ...currentCatalog,
      models: [
        {
          ...currentCatalog.models[0],
          selected: true,
        },
      ],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(catalog)),
    );

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    await user.click(screen.getByTestId("admin-price-book-unused-toggle"));
    expect(screen.getByText("No unused model rates for this provider.")).toBeInTheDocument();
  });

  it("sorts model and machine rates from the column header", async () => {
    const catalog = {
      ...currentCatalog,
      models: [
        { ...currentCatalog.models[0], match_key: "beta-model", input_cents_per_million: 100, selected: true },
        { ...currentCatalog.models[0], match_key: "alpha-model", input_cents_per_million: 500, selected: true },
      ],
      vms: [
        { match_key: "e1-small-amd64", match_mode: "exact", micros_per_second: 40 },
        { match_key: "e1-large-amd64", match_mode: "exact", micros_per_second: 70 },
      ],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(catalog)),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("alpha-model")).toBeInTheDocument();

    const bodyRows = () => screen.getAllByRole("row").slice(1);
    expect(bodyRows()[0]).toHaveTextContent("alpha-model");
    expect(screen.getByRole("columnheader", { name: "Model" })).toHaveAttribute("aria-sort", "ascending");

    await user.click(screen.getByRole("button", { name: "Input" }));
    expect(screen.getByRole("columnheader", { name: "Input" })).toHaveAttribute("aria-sort", "ascending");
    expect(bodyRows()[0]).toHaveTextContent("beta-model");

    await user.click(screen.getByRole("button", { name: "Input" }));
    expect(screen.getByRole("columnheader", { name: "Input" })).toHaveAttribute("aria-sort", "descending");
    expect(bodyRows()[0]).toHaveTextContent("alpha-model");

    await user.click(screen.getByRole("tab", { name: "Machines" }));
    expect(bodyRows()[0]).toHaveTextContent("e1-large-amd64");

    await user.click(screen.getByRole("button", { name: "Micros per second" }));
    expect(bodyRows()[0]).toHaveTextContent("e1-small-amd64");

    await user.click(screen.getByRole("button", { name: "Micros per second" }));
    expect(bodyRows()[0]).toHaveTextContent("e1-large-amd64");
  });

  it("disables update on Anthropic and OpenAI tabs", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(currentCatalog)),
    );

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByTestId("admin-price-book-sync")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Anthropic" }));
    expect(screen.getByTestId("admin-price-book-sync-disabled")).toBeDisabled();
    expect(screen.getByText("The Anthropic API does not publish prices.")).toBeInTheDocument();
  });
});
