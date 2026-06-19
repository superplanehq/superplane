import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SignupWaitlist } from "./SignupWaitlist";

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION?: string;
  hbspt?: {
    forms?: {
      create: ReturnType<typeof vi.fn>;
    };
  };
};

const waitlistWindow = window as SignupWaitlistWindow;

afterEach(() => {
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION;
  delete waitlistWindow.hbspt;
  document.getElementById("hubspot-forms-script")?.remove();
});

describe("SignupWaitlist", () => {
  it("does not render the HubSpot form without complete config", () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";

    const { container } = render(<SignupWaitlist />);

    expect(container.querySelector("#signup-waitlist-hubspot-form")).toBeNull();
    expect(screen.queryByText(/Leave your email/)).toBeNull();
    expect(document.getElementById("hubspot-forms-script")).toBeNull();
  });

  it("renders the HubSpot form when portal and form IDs are configured", async () => {
    const createForm = vi.fn();
    waitlistWindow.hbspt = { forms: { create: createForm } };
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION = "eu1";

    const { container } = render(<SignupWaitlist />);

    expect(container.querySelector("#signup-waitlist-hubspot-form")).toBeInTheDocument();
    expect(screen.getByText(/Leave your email/)).toBeInTheDocument();
    await waitFor(() => {
      expect(createForm).toHaveBeenCalledWith({
        portalId: "portal-1",
        formId: "form-1",
        target: "#signup-waitlist-hubspot-form",
        region: "eu1",
      });
    });
  });

  it("loads the HubSpot script when the forms client is not ready", () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";

    render(<SignupWaitlist />);

    expect(document.getElementById("hubspot-forms-script")).toHaveAttribute(
      "src",
      "https://js.hsforms.net/forms/embed/v2.js",
    );
  });
});
