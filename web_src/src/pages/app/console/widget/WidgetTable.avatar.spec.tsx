import { render } from "@testing-library/react";
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

describe("WidgetTable avatar column — direct image URLs", () => {
  it("uses a direct image URL as the avatar source instead of treating it as a username", () => {
    const view = renderAvatar(AVATAR_RENDER, [{ id: "row-1", name: "Ada", avatarUrl: "https://example.com/ada.png" }]);
    const img = view.container.querySelector('table tbody tr td:nth-child(2) [data-slot="avatar"] img');
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("https://example.com/ada.png");
    view.unmount();
  });

  it("resolves {{ expr }} field values into the image src", () => {
    const view = renderAvatar(
      {
        kind: "table",
        columns: [{ field: '{{ "https://cdn.example.com/u/" + userId + ".jpg" }}', label: "Avatar", format: "avatar" }],
      },
      [{ id: "row-1", userId: "7" }],
    );
    const img = view.container.querySelector('table tbody tr td [data-slot="avatar"] img');
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
});
