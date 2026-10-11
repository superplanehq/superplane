import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FleetManagerStep } from "./FleetManagerStep";
import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

const yaml = "id: self-host\nsuperplaneUrl: https://example\n";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("FleetManagerStep", () => {
  it("shows Fleet Manager YAML after prepare", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () => new Response(JSON.stringify({ yaml, fleet_id: "e1-large-amd64", token: "tok" }), { status: 200 }),
      ),
    );

    render(<FleetManagerStep onContinue={vi.fn()} onSkip={vi.fn()} />);

    expect(await screen.findByTestId("owner-setup-fleet-yaml")).toHaveValue(yaml);
    expect(screen.getByRole("button", { name: OWNER_SETUP_COPY.fleet.continue })).toBeEnabled();
  });

  it("skips Fleet Manager after the YAML loads", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ yaml, token: "tok" }), { status: 200 })),
    );
    const onSkip = vi.fn();

    render(<FleetManagerStep onContinue={vi.fn()} onSkip={onSkip} />);
    await screen.findByTestId("owner-setup-fleet-yaml");
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    expect(onSkip).toHaveBeenCalledTimes(1);
  });

  it("lets the operator skip when prepare fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("", { status: 500 })),
    );
    const onSkip = vi.fn();

    render(<FleetManagerStep onContinue={vi.fn()} onSkip={onSkip} />);

    await waitFor(() => expect(screen.getByText(OWNER_SETUP_COPY.fleet.error)).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));
    expect(onSkip).toHaveBeenCalledTimes(1);
  });
});
