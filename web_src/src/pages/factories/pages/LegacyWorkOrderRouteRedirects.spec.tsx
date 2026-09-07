import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it } from "vitest";

import { LegacyWorkOrderPermalinkRedirect, LegacyWorkOrdersRedirect } from "./LegacyWorkOrderRouteRedirects";

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path=":organizationId/workspaces/:factoryKey">
          <Route path="work-orders/*" element={<LegacyWorkOrdersRedirect />} />
          <Route path="work-order/:orderNumber" element={<LegacyWorkOrderPermalinkRedirect />} />
        </Route>
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

describe("LegacyWorkOrdersRedirect", () => {
  it("sends the bare list to the tasks list", () => {
    renderAt("/org-1/workspaces/SP/work-orders");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/tasks");
  });

  it("forwards the new child segment", () => {
    renderAt("/org-1/workspaces/SP/work-orders/new");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/tasks/new");
  });

  it("forwards an id segment and the query string", () => {
    renderAt("/org-1/workspaces/SP/work-orders/order-uuid?foo=bar");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/tasks/order-uuid?foo=bar");
  });
});

describe("LegacyWorkOrderPermalinkRedirect", () => {
  it("sends the singular permalink to the task permalink", () => {
    renderAt("/org-1/workspaces/SP/work-order/42");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/task/42");
  });

  it("preserves the query string", () => {
    renderAt("/org-1/workspaces/SP/work-order/42?lineId=line-hotfix");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/task/42?lineId=line-hotfix");
  });
});
