import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";
import { OwnerTimeCostRow } from "../work-order-popup-redesign/popupShared";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";

function PopupOwner({ fixture }: { fixture: ReturnType<typeof splitRunFixtureForWorkOrder> }) {
  const edits = useSplitRunWorkOrderEdits({
    title: fixture.title,
    description: "",
    owner: fixture.owner,
    assigneeIds: [fixture.owner.id],
    footerKind: "draft",
  });
  return <OwnerTimeCostRow fixture={{ ...fixture, owner: edits.owner }} />;
}

// The roster may be missing while loading; only explicit API status marks deletion.
describe("task deleted-member indicator", () => {
  it.each([
    { deleted: false, formerMember: false, label: null },
    { deleted: true, formerMember: false, label: "Deleted user" },
    { deleted: false, formerMember: true, label: "Former member" },
  ])("shows the API status $label", ({ deleted, formerMember, label }) => {
    const member = { id: "member", name: "Jamie Verification", deleted, formerMember };
    const fixture = splitRunFixtureForWorkOrder({
      id: "task",
      title: "Member verification",
      state: "STATE_DRAFT",
      number: "111",
      assignees: [member],
      createdBy: { user: member },
    });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <PopupOwner fixture={fixture} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("Jamie Verification");
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("JV");
    for (const status of ["Deleted user", "Former member"]) {
      if (status === label) {
        expect(screen.getByText(status)).toBeInTheDocument();
      } else {
        expect(screen.queryByText(status)).not.toBeInTheDocument();
      }
    }
  });
});
