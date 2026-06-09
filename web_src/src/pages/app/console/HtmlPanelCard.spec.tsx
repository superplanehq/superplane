import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { canvasKeys, type CanvasMemoryEntry, type ConsolePanel } from "@/hooks/useCanvasData";

import { ConsoleContextProvider } from "./ConsoleContextProvider";
import { HtmlPanelCard } from "./HtmlPanelCard";

function renderHtml(body: string) {
  return renderWithVariables({
    panel: { id: "html-test", type: "html", content: { body } },
  });
}

interface RenderWithVariablesOptions {
  panel: ConsolePanel;
  memoryEntries?: CanvasMemoryEntry[];
}

function renderWithVariables({ panel, memoryEntries = [] }: RenderWithVariablesOptions) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  queryClient.setQueryData(canvasKeys.canvasMemoryEntries("canvas-1"), memoryEntries);
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <HtmlPanelCard panel={panel} readOnly onDelete={() => {}} onChange={() => {}} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("HtmlPanelCard rendering", () => {
  it("renders plain authored HTML inside the scoped root", () => {
    renderHtml('<section><h2>Status</h2><p class="text-slate-800">All clear.</p></section>');
    const view = screen.getByTestId("console-html");
    expect(view.querySelector("section")).not.toBeNull();
    expect(view.querySelector("h2")?.textContent).toBe("Status");
    expect(view.querySelector("p")?.textContent).toBe("All clear.");
    // The root carries the scope marker that <style> selectors anchor to.
    expect(view.hasAttribute("data-console-html-root")).toBe(true);
  });

  it("strips <script> tags from authored HTML", () => {
    renderHtml("Hello <script>window.__pwned = true;</script><p>world</p>");
    const view = screen.getByTestId("console-html");
    expect(view.querySelector("script")).toBeNull();
    expect(view.textContent).toMatch(/Hello/);
    expect(view.textContent).toMatch(/world/);
  });

  it("strips inline event handlers from allowed tags", () => {
    renderHtml('<a href="https://example.com" onclick="alert(1)">link</a>');
    const view = screen.getByTestId("console-html");
    const anchor = view.querySelector("a");
    expect(anchor).not.toBeNull();
    expect(anchor!.getAttribute("onclick")).toBeNull();
  });

  it("allows http(s) image sources but rejects javascript: URLs", () => {
    renderHtml('<img src="https://cdn.example.com/x.png" alt="ok"><img src="javascript:alert(1)" alt="bad">');
    const view = screen.getByTestId("console-html");
    const imgs = view.querySelectorAll("img");
    expect(imgs).toHaveLength(2);
    expect(imgs[0].getAttribute("src")).toBe("https://cdn.example.com/x.png");
    expect(imgs[1].getAttribute("src")).toBeNull();
  });
});

describe("HtmlPanelCard variable interpolation", () => {
  it("interpolates memory variables before sanitization", async () => {
    const panel: ConsolePanel = {
      id: "panel-1",
      type: "html",
      content: {
        title: "Latest {{ rec.service }}",
        body: "<p>Status: <strong>{{ rec.status }}</strong></p>",
        variables: [{ name: "rec", source: { kind: "memory", namespace: "deploys" } }],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        {
          id: "row-old",
          namespace: "deploys",
          values: { service: "api", status: "passed" },
          source: "node",
          createdAt: "2026-06-01T00:00:00Z",
        },
        {
          id: "row-new",
          namespace: "deploys",
          values: { service: "web", status: "failed" },
          source: "node",
          createdAt: "2026-06-04T00:00:00Z",
        },
      ],
    });
    const view = await waitFor(() => screen.getByTestId("console-html"));
    expect(view.textContent).toMatch(/Status: failed/);
    expect(screen.getByText(/Latest web/)).toBeTruthy();
  });

  it("sanitizes script payloads injected via interpolated values", async () => {
    // A memory value that happens to contain markup must not bypass the
    // sanitizer just because it arrives through `{{ }}`.
    const panel: ConsolePanel = {
      id: "panel-xss",
      type: "html",
      content: {
        body: "<p>{{ rec.snippet }}</p>",
        variables: [{ name: "rec", source: { kind: "memory", namespace: "deploys" } }],
      },
    };
    renderWithVariables({
      panel,
      memoryEntries: [
        {
          id: "row",
          namespace: "deploys",
          values: { snippet: "<script>window.__pwned = true</script>safe" },
          source: "node",
        },
      ],
    });
    const view = await waitFor(() => screen.getByTestId("console-html"));
    expect(view.querySelector("script")).toBeNull();
    expect(view.textContent).toMatch(/safe/);
  });
});
