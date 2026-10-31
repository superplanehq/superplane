import { render, screen } from "@testing-library/react";
import { expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { CodexSetupFailure } from "./CodexSetupFailure";

vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));

it("uses the existing credit recovery action for a hosted setup rejection", () => {
  render(
    <MemoryRouter>
      <CodexSetupFailure organizationId="org-1" error={{ message: "This organization has no hosted credit." }} />
    </MemoryRouter>,
  );
  expect(screen.getByRole("alert")).toHaveTextContent("The organization has no SuperPlane hosted credit.");
  expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute("href", "/org-1/organization/billing");
});
