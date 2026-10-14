import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FleetManagerStep } from "./FleetManagerStep";
import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

afterEach(() => vi.unstubAllGlobals());

describe("FleetManagerStep", () => {
  it("links to the reusable setup page without making a request", () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);

    render(<FleetManagerStep onContinue={vi.fn()} onSkip={vi.fn()} />);

    expect(screen.getByRole("link", { name: OWNER_SETUP_COPY.fleet.open })).toHaveAttribute(
      "href",
      "/admin/fleet-manager-setup",
    );
    expect(fetch).not.toHaveBeenCalled();
  });

  it("lets the administrator finish or skip setup", () => {
    const onContinue = vi.fn();
    const onSkip = vi.fn();
    render(<FleetManagerStep onContinue={onContinue} onSkip={onSkip} />);

    fireEvent.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue }));
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    expect(onContinue).toHaveBeenCalledTimes(1);
    expect(onSkip).toHaveBeenCalledTimes(1);
  });
});
