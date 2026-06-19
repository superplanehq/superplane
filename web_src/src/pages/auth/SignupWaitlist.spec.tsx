import { render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { SignupWaitlist } from "./SignupWaitlist";

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID?: string;
  ml?: unknown;
};

const waitlistWindow = window as SignupWaitlistWindow;

afterEach(() => {
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID;
  delete waitlistWindow.ml;
  document.getElementById("mailerlite-universal-script")?.remove();
});

describe("SignupWaitlist", () => {
  it("does not render the MailerLite form without complete config", () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID = "account-1";

    const { container } = render(<SignupWaitlist />);

    expect(container.querySelector(".ml-embedded")).toBeNull();
    expect(document.getElementById("mailerlite-universal-script")).toBeNull();
  });

  it("renders the MailerLite form when account and form IDs are configured", async () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID = "account-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID = "form-1";

    const { container } = render(<SignupWaitlist />);

    expect(container.querySelector(".ml-embedded")).toHaveAttribute("data-form", "form-1");
    await waitFor(() => {
      expect(document.getElementById("mailerlite-universal-script")).toHaveAttribute(
        "src",
        "https://assets.mailerlite.com/js/universal.js",
      );
    });
  });
});
