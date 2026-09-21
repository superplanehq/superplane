import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

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

const catalogFor = (version: string, matchKey: string, current = "2026-09-09.1") => ({
  current_version: current,
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
const savedCatalog = {
  ...catalogFor("2026-09-13.1", "claude-sonnet", "2026-09-13.1"),
  versions: [
    { version: "2026-09-13.1", effective_at: "2026-09-13T00:00:00Z", created_at: "2026-09-13T00:00:00Z" },
    ...versions,
  ],
};

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
  vi.mocked(showSuccessToast).mockClear();
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

    await user.click(screen.getByRole("tab", { name: "VMs" }));
    await user.type(screen.getByLabelText("Machine type", { selector: "#price-book-add-vm-key" }), "e1-test-amd64");
    await user.click(screen.getByRole("button", { name: "Add VM rate" }));
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
        if (init?.method === "POST" && String(input) === "/admin/api/price-books/sync") {
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

  it("renames and removes VM rates on the current catalog", async () => {
    const renamedVm = "e1-xlarge-amd64";
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
        expect(body.vms).toEqual([{ match_key: renamedVm, match_mode: "exact", micros_per_second: 70 }]);
        return jsonResponse(savedCatalog);
      }
      return jsonResponse(catalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("claude-sonnet")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "VMs" }));
    const machineType = screen.getByDisplayValue("e1-large-amd64");
    await user.clear(machineType);
    await user.type(machineType, " E1-XLARGE-AMD64 ");
    await user.tab();
    expect(machineType).toHaveValue(renamedVm);

    const removeButtons = screen.getAllByRole("button", { name: "Remove" });
    await user.click(removeButtons[1]);
    expect(screen.queryByDisplayValue("e1-small-amd64")).not.toBeInTheDocument();

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

    await user.click(screen.getByRole("tab", { name: "VMs" }));
    const machineType = screen.getByDisplayValue("e1-large-amd64");

    await user.clear(machineType);
    await user.tab();
    expect(showErrorToast).toHaveBeenCalledWith("Enter a machine type.");
    expect(machineType).toHaveValue("e1-large-amd64");

    await user.clear(machineType);
    await user.type(machineType, "   ");
    await user.tab();
    expect(showErrorToast).toHaveBeenCalledWith("Enter a machine type.");
    expect(machineType).toHaveValue("e1-large-amd64");

    await user.clear(machineType);
    await user.type(machineType, "e1-small-amd64");
    await user.tab();
    expect(showErrorToast).toHaveBeenCalledWith("That VM rate already exists.");
    expect(machineType).toHaveValue("e1-large-amd64");

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

    await user.click(screen.getByRole("tab", { name: "VMs" }));
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(screen.getByText("This version has no VM rates.")).toBeInTheDocument();
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

    await user.click(screen.getByRole("tab", { name: "VMs" }));
    expect(screen.getByText("e1-large-amd64")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("e1-large-amd64")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();
  });

  it("shows selected model in selected group and unselected in other group", async () => {
    const catalog = {
      ...currentCatalog,
      models: [
        {
          ...currentCatalog.models[0],
          selected: true,
        },
        {
          match_key: "other-model",
          match_mode: "prefix",
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

    renderPage();

    expect(await screen.findByText("Selected models")).toBeInTheDocument();
    expect(screen.getByText("Other models")).toBeInTheDocument();
    expect(screen.getByText("claude-sonnet")).toBeInTheDocument();
    expect(screen.getByText("other-model")).toBeInTheDocument();
  });

  it("shows empty state when no model is selected", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(currentCatalog)),
    );

    renderPage();

    expect(await screen.findByText("Selected models")).toBeInTheDocument();
    expect(screen.getByText("No models are selected in hosted LLM settings.")).toBeInTheDocument();
  });

  it("edits a rate in the other models group", async () => {
    const catalog = {
      ...currentCatalog,
      models: [
        {
          ...currentCatalog.models[0],
          selected: true,
        },
        {
          match_key: "other-model",
          match_mode: "prefix",
          input_cents_per_million: 100,
          output_cents_per_million: 500,
          cache_read_cents_per_million: 10,
          cache_write_cents_per_million: 50,
          reasoning_cents_per_million: 0,
          selected: false,
        },
      ],
    };
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT" && String(input) === "/admin/api/price-books") {
        const body = JSON.parse(String(init.body)) as {
          models: { input_cents_per_million: number }[];
        };
        expect(body.models[1].input_cents_per_million).toBe(200);
        return jsonResponse(savedCatalog);
      }
      return jsonResponse(catalog);
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("other-model")).toBeInTheDocument();
    const inputs = screen.getAllByDisplayValue("1.00");
    const otherInput = inputs.find((input) => input.closest("tr")?.textContent?.includes("other-model"));
    expect(otherInput).toBeTruthy();
    await user.clear(otherInput!);
    await user.type(otherInput!, "2");
    await user.tab();

    await user.click(screen.getByTestId("admin-price-book-save"));
    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalled();
    });
  });
});
