import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SignupWaitlist } from "./SignupWaitlist";

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION?: string;
};

const waitlistWindow = window as SignupWaitlistWindow;

afterEach(() => {
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION;
  document.cookie = "hubspotutk=; Max-Age=0; path=/";
  vi.unstubAllGlobals();
});

describe("SignupWaitlist", () => {
  it("does not render the notify form without complete HubSpot config", () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";

    render(<SignupWaitlist />);

    expect(screen.queryByLabelText("Email")).toBeNull();
    expect(screen.queryByRole("button", { name: "Notify me" })).toBeNull();
    expect(screen.queryByText(/Leave your email/)).toBeNull();
  });

  it("submits the native notify form to HubSpot", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    document.cookie = "hubspotutk=visitor-1; path=/";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION = "eu1";

    render(<SignupWaitlist />);

    await userEvent.type(screen.getByLabelText("Email"), " person@example.com ");
    await userEvent.click(screen.getByRole("button", { name: "Notify me" }));

    await waitFor(() => {
      expect(screen.getByRole("status")).toHaveTextContent("You are on the waitlist");
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "https://api-eu1.hsforms.com/submissions/v3/integration/submit/portal-1/form-1",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
      }),
    );

    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body.fields).toEqual([{ name: "email", value: "person@example.com" }]);
    expect(body.context).toEqual(
      expect.objectContaining({
        hutk: "visitor-1",
        pageUri: expect.any(String),
      }),
    );
  });

  it("shows a retryable error when HubSpot rejects the submission", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 500 })));
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";

    render(<SignupWaitlist />);

    await userEvent.type(screen.getByLabelText("Email"), "person@example.com");
    await userEvent.click(screen.getByRole("button", { name: "Notify me" }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("We could not save your email");
    });
  });
});
