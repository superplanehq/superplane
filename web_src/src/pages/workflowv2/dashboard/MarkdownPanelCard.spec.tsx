import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { DashboardPanel } from "@/hooks/useCanvasData";

import { MarkdownPanelCard } from "./MarkdownPanelCard";

function renderMarkdown(body: string) {
  const panel: DashboardPanel = {
    id: "md-test",
    type: "markdown",
    content: { body },
  };
  return render(<MarkdownPanelCard panel={panel} readOnly onDelete={() => {}} onChange={() => {}} />);
}

describe("MarkdownPanelCard rendering", () => {
  it("renders a GFM pipe table from hand-written markdown", () => {
    renderMarkdown("| Service | Status |\n| --- | --- |\n| api | passed |\n| web | failed |\n");

    const view = screen.getByTestId("dashboard-markdown");
    const table = view.querySelector("table");
    expect(table).not.toBeNull();
    const headers = table!.querySelectorAll("th");
    expect(headers).toHaveLength(2);
    expect(headers[0].textContent).toBe("Service");
    expect(headers[1].textContent).toBe("Status");
    const rows = table!.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(2);
    expect(rows[0].textContent).toMatch(/api/);
    expect(rows[1].textContent).toMatch(/failed/);
  });

  it("preserves <details>/<summary> accordions and the open attribute", () => {
    renderMarkdown("<details open>\n<summary>Troubleshooting</summary>\n\nFlush the cache.\n\n</details>");

    const view = screen.getByTestId("dashboard-markdown");
    const details = view.querySelector("details");
    expect(details).not.toBeNull();
    expect(details!.hasAttribute("open")).toBe(true);
    const summary = details!.querySelector("summary");
    expect(summary?.textContent).toBe("Troubleshooting");
    expect(details!.textContent).toMatch(/Flush the cache/);
  });

  it("strips unsafe raw HTML like <script> tags", () => {
    renderMarkdown("Hello <script>window.__pwned = true;</script> world");

    const view = screen.getByTestId("dashboard-markdown");
    expect(view.querySelector("script")).toBeNull();
    expect(view.textContent).toMatch(/Hello/);
    expect(view.textContent).toMatch(/world/);
  });

  it("strips inline event handlers from allowed tags", () => {
    renderMarkdown('<a href="https://example.com" onclick="alert(1)">link</a>');

    const view = screen.getByTestId("dashboard-markdown");
    const anchor = view.querySelector("a");
    expect(anchor).not.toBeNull();
    expect(anchor!.getAttribute("onclick")).toBeNull();
  });
});
