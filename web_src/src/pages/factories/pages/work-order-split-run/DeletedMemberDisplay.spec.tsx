import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";
import { OwnerTimeCostRow } from "../work-order-popup-redesign/popupShared";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

// The roster may be missing while loading; only explicit API status marks deletion.
describe("task deleted-member indicator", () => {
  it.each([false, true])("shows deleted status only when the API reports %s", (deleted) => {
    const member = { id: "member", name: "Jamie Verification", deleted };
    const fixture = splitRunFixtureForWorkOrder({
      id: "task",
      title: "Member verification",
      state: "STATE_DRAFT",
      number: "111",
      assignees: [member],
      createdBy: { user: member },
    });
    render(
      <MemoryRouter>
        <OwnerTimeCostRow fixture={fixture} />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("Jamie Verification");
    expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent("JV");
    if (deleted) {
      expect(screen.getByText("Deleted user")).toBeInTheDocument();
    } else {
      expect(screen.queryByText("Deleted user")).not.toBeInTheDocument();
    }
  });
});
