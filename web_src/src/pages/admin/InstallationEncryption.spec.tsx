import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import InstallationEncryption from "./InstallationEncryption";

describe("InstallationEncryption", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("restores a failed migration and blocks rotation until it completes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const body = String(input).endsWith("/migrations")
          ? {
              migrations: [
                {
                  id: "job-1",
                  targetKeyId: "key-1",
                  state: "failed",
                  storageLocation: "secrets.data",
                  errorMessage: "Could not decrypt a value.",
                },
              ],
            }
          : { keys: [{ id: "key-1", masterKeyRef: "local-v1" }], activeKeyId: "key-1" };
        return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
      }),
    );

    render(<InstallationEncryption />);

    expect(await screen.findByText("Could not decrypt a value.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rotate data key" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Start migration" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Retry migration" })).toBeEnabled();
  });
});
