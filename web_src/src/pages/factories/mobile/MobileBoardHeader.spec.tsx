import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  REFUND_FACTORY,
  REFUND_FACTORY_LINES,
  REFUND_LINE_PLAN_ID,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { MobileBoardHeader } from "./MobileBoardHeader";

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrdersPage: () => ({ data: [], isLoading: false }),
  useWorkOrderArtifacts: () => ({ data: [], isLoading: false }),
  useSendWorkOrderToBacklog: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
const permissions = { create: true };
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => permissions.create }) }));
const nextFactory = { ...REFUND_FACTORY, id: "next-workspace", name: "Engineering", key: "ENG", urlId: "abcd1234" };

function PhoneBoardHeader() {
  const state = useWorkOrderListState("factory-1");
  const location = useLocation();
  return (
    <>
      <MobileBoardHeader
        state={state}
        searchRef={createRef()}
        sourceOptions={[]}
        assigneeOptions={[]}
        showPullRequestMerge={false}
        lines={REFUND_FACTORY_LINES}
        lineId={REFUND_LINE_PLAN_ID}
        onSelectLine={vi.fn()}
        organizationId="org-1"
        factoryId="factory-1"
        factoryKey={PRIMARY_FACTORY_ROUTE_SEGMENT}
        canManageClosedStatus={false}
      />
      <p data-testid="location">{location.pathname}</p>
    </>
  );
}

function renderHeader(path = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: REFUND_FACTORY.id!,
          factoryKey: REFUND_FACTORY.key!,
          routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
          factory: REFUND_FACTORY,
          factories: [REFUND_FACTORY, nextFactory],
          openCreateWorkOrder: vi.fn(),
        }}
      >
        <PhoneBoardHeader />
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("MobileBoardHeader", () => {
  beforeEach(() => {
    permissions.create = true;
    window.localStorage.clear();
  });

  it("searches workspace choices and opens the new workspace board", async () => {
    const user = userEvent.setup();
    renderHeader();
    expect(screen.getByRole("heading", { name: "Board" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Filter" })).toHaveTextContent("Filter");
    await user.click(screen.getByRole("button", { name: /Switch workspace,/ }));
    const sheet = screen.getByRole("dialog", { name: "Switch workspace" });
    expect(within(sheet).getByRole("button", { name: /Current workspace/ })).toHaveAttribute("aria-current", "true");
    await user.type(within(sheet).getByRole("textbox", { name: "Search workspaces" }), "engineering");
    expect(within(sheet).queryByText("Current workspace")).not.toBeInTheDocument();
    await user.click(within(sheet).getByRole("button", { name: "Engineering" }));
    expect(screen.getByTestId("location")).toHaveTextContent(
      `/org-1/workspaces/eng-abcd1234/lines/${REFUND_LINE_PLAN_ID}`,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("preserves Velocity when switching and hides creation without permission", async () => {
    permissions.create = false;
    const user = userEvent.setup();
    renderHeader(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/velocity`);
    await user.click(screen.getByRole("button", { name: /Switch workspace,/ }));
    expect(screen.queryByRole("button", { name: "Create workspace" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Engineering" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/eng-abcd1234/velocity");
  });

  it("opens workspace creation from the sheet", async () => {
    const user = userEvent.setup();
    renderHeader();
    await user.click(screen.getByRole("button", { name: /Switch workspace,/ }));
    await user.click(screen.getByRole("button", { name: "Create workspace" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/new");
  });
});
