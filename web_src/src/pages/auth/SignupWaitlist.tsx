import type React from "react";
import { useEffect } from "react";
import { Text } from "@/components/Text/text";
import { getSignupWaitlistConfig } from "@/lib/signupWaitlistConfig";

const hubSpotScriptID = "hubspot-forms-script";
const hubSpotFormTargetID = "signup-waitlist-hubspot-form";

type HubSpotFormOptions = {
  portalId: string;
  formId: string;
  target: string;
  region?: string;
};

type HubSpotWindow = Window & {
  hbspt?: {
    forms?: {
      create: (options: HubSpotFormOptions) => void;
    };
  };
};

const loadHubSpotScript = () => {
  const existingScript = document.getElementById(hubSpotScriptID) as HTMLScriptElement | null;
  if (existingScript) {
    return existingScript;
  }

  const script = document.createElement("script");
  script.id = hubSpotScriptID;
  script.async = true;
  script.src = "https://js.hsforms.net/forms/embed/v2.js";
  document.head.appendChild(script);
  return script;
};

export const SignupWaitlist: React.FC = () => {
  const hubSpotConfig = getSignupWaitlistConfig();
  const hubSpotPortalID = hubSpotConfig?.portalID;
  const hubSpotFormID = hubSpotConfig?.formID;
  const hubSpotRegion = hubSpotConfig?.region;
  const hasHubSpotForm = Boolean(hubSpotPortalID && hubSpotFormID);

  useEffect(() => {
    if (!hubSpotPortalID || !hubSpotFormID) {
      return;
    }

    const renderForm = () => {
      const forms = (window as HubSpotWindow).hbspt?.forms;
      if (!forms) {
        return;
      }

      forms.create({
        portalId: hubSpotPortalID,
        formId: hubSpotFormID,
        target: `#${hubSpotFormTargetID}`,
        ...(hubSpotRegion ? { region: hubSpotRegion } : {}),
      });
    };

    if ((window as HubSpotWindow).hbspt?.forms) {
      renderForm();
      return;
    }

    const script = loadHubSpotScript();
    script.addEventListener("load", renderForm);

    return () => script.removeEventListener("load", renderForm);
  }, [hubSpotFormID, hubSpotPortalID, hubSpotRegion]);

  return (
    <div className="space-y-4">
      <Text className="text-left text-sm leading-6 text-gray-600">
        We are opening access gradually while demand is high.
        {hasHubSpotForm && " Leave your email and we will send an invite as soon as capacity is available."}
      </Text>

      {hasHubSpotForm && <div id={hubSpotFormTargetID} className="-mx-5" />}
    </div>
  );
};
