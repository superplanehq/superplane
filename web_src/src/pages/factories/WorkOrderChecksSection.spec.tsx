import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "./__fixtures__/factoryPageResponses";
import { BOOLEAN_CHECK_CI_PASS, BOOLEAN_CHECK_SECURITY_SCAN_FAIL } from "./__fixtures__/workOrderCheckFixtures";
import { WorkOrderChecksSection } from "./WorkOrderChecksSection";

describe("WorkOrderChecksSection boolean checks", () => {
  it("renders a Pass/Fail badge (not a score) for boolean checks", () => {
    render(
      <MemoryRouter>
        <WorkOrderChecksSection
          checks={[BOOLEAN_CHECK_CI_PASS, BOOLEAN_CHECK_SECURITY_SCAN_FAIL]}
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={PRIMARY_FACTORY_KEY}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId(`work-order-check-${BOOLEAN_CHECK_CI_PASS.id}`)).toHaveTextContent("Pass");
    expect(screen.getByTestId(`work-order-check-${BOOLEAN_CHECK_SECURITY_SCAN_FAIL.id}`)).toHaveTextContent("Fail");
  });

  it("opens the check dialog with pass/fail styling and no score line when a boolean card is clicked", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <WorkOrderChecksSection
          checks={[BOOLEAN_CHECK_SECURITY_SCAN_FAIL]}
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={PRIMARY_FACTORY_KEY}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId(`work-order-check-${BOOLEAN_CHECK_SECURITY_SCAN_FAIL.id}`));

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Fail");
    expect(dialog).toHaveTextContent(BOOLEAN_CHECK_SECURITY_SCAN_FAIL.summary ?? "");
    expect(dialog).not.toHaveTextContent("Previous run");
  });
});
