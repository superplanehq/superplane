import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { OWNER_SETUP_COPY } from "./ownerSetupCopy";
import { SmtpStep } from "./SmtpStep";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("SmtpStep", () => {
  it("skips SMTP without saving", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const onSkip = vi.fn();

    render(<SmtpStep onContinue={vi.fn()} onSkip={onSkip} />);
    fireEvent.click(screen.getByTestId("owner-setup-smtp-skip"));

    expect(onSkip).toHaveBeenCalledTimes(1);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("saves SMTP and continues", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => new Response("{}", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const onContinue = vi.fn();

    render(<SmtpStep onContinue={onContinue} onSkip={vi.fn()} />);
    await user.type(screen.getByTestId("owner-setup-smtp-host"), "smtp.example.com");
    await user.type(screen.getByTestId("owner-setup-smtp-from-email"), "noreply@example.com");
    await user.click(screen.getByRole("button", { name: OWNER_SETUP_COPY.smtp.save }));

    await waitFor(() => expect(onContinue).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith(
      "/admin/api/installation/network-settings",
      expect.objectContaining({
        method: "PATCH",
        body: expect.stringContaining('"smtp_enabled":true'),
      }),
    );
  });
});
