import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { billingState } from "./FirstRunSetup.chrome.spec.billing";
import { FirstRunSetup } from "./FirstRunSetup";
import type { OnboardingPageModel } from "./useFirstRunSetupFlow";

export function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  billingState.byok = model.bringYourOwnKey;
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}
