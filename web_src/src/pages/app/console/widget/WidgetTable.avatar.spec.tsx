import { render, screen, fireEvent, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect } from "vitest";

import { ConsoleContextProvider } from "../ConsoleContextProvider";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

function renderAvatar(tableRender: WidgetTableRender, rows: unknown[]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ConsoleContextProvider canvasId="canvas-1" organizationId="org-1" nodes={[]} canRunNodes={false}>
          <WidgetTable render={tableRender} rows={rows} isLoading={false} />
        </ConsoleContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const AVATAR_RENDER: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "name", label: "Name" },
    { field: "avatarUrl", label: "Avatar", format: "avatar" },
  ],
};

describe("WidgetTable avatar column", () => {
  it("renders a circular <img> with the row URL and lazy loading", () => {
    const view = renderAvatar(AVATAR_RENDER, [{ id: "row-1", name: "Ada", avatarUrl: "https://example.com/ada.png" }]);
    const img = view.container.querySelector("table tbody tr td:nth-child(2) img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("https://example.com/ada.png");
    expect(img!.getAttribute("alt")).toBe("Avatar");
    expect(img!.getAttribute("loading")).toBe("lazy");
    expect(img!.getAttribute("referrerpolicy")).toBe("no-referrer");
    expect(img!.className).toContain("rounded-full");
    expect(img!.className).toContain("object-cover");
    view.unmount();
  });

  it("resolves {{ expr }} field values into the image src", () => {
    const view = renderAvatar(
      {
        kind: "table",
        columns: [{ field: "{{ profile.imageUrl }}", label: "Avatar", format: "avatar" }],
      },
      [{ id: "row-1", profile: { imageUrl: "https://cdn.example.com/u/7.jpg" } }],
    );
    const img = view.container.querySelector("table tbody tr td img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("https://cdn.example.com/u/7.jpg");
    view.unmount();
  });

  it("renders an em-dash placeholder when the URL is missing or blank", () => {
    const view = renderAvatar(AVATAR_RENDER, [{ id: "row-1", name: "Ada", avatarUrl: "" }]);
    const cell = view.container.querySelector("table tbody tr td:nth-child(2)");
    expect(cell).not.toBeNull();
    expect(cell!.querySelector("img")).toBeNull();
    expect(cell!.textContent).toBe("—");
    view.unmount();
  });

  it("falls back to a neutral disc when the image errors out", async () => {
    const view = renderAvatar(AVATAR_RENDER, [
      { id: "row-1", name: "Ada", avatarUrl: "https://example.com/broken.png" },
    ]);
    const img = view.container.querySelector("table tbody tr td:nth-child(2) img")!;
    await act(async () => {
      fireEvent.error(img);
    });
    expect(view.container.querySelector("table tbody tr td:nth-child(2) img")).toBeNull();
    const fallback = screen.getByTestId("widget-avatar-fallback");
    expect(fallback.className).toContain("rounded-full");
    expect(fallback.getAttribute("aria-label")).toBe("Avatar");
    view.unmount();
  });
});
