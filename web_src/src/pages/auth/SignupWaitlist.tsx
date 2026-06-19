import type React from "react";
import { useEffect } from "react";
import { Text } from "@/components/Text/text";

const mailerLiteScriptID = "mailerlite-universal-script";

type MailerLiteConfig = {
  accountID: string;
  formID: string;
};

type MailerLiteClient = {
  (...args: unknown[]): void;
  q?: unknown[][];
};

type MailerLiteWindow = Window & {
  ml?: MailerLiteClient;
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID?: string;
};

const getMailerLiteConfig = (): MailerLiteConfig | null => {
  const win = window as MailerLiteWindow;
  const accountID = win.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_ACCOUNT_ID?.trim();
  const formID = win.SUPERPLANE_SIGNUP_WAITLIST_MAILERLITE_FORM_ID?.trim();

  if (!accountID || !formID) {
    return null;
  }

  return { accountID, formID };
};

const ensureMailerLiteClient = () => {
  const win = window as MailerLiteWindow;
  if (win.ml) {
    return win.ml;
  }

  const queuedClient: MailerLiteClient = (...args: unknown[]) => {
    queuedClient.q = queuedClient.q || [];
    queuedClient.q.push(args);
  };

  win.ml = queuedClient;
  return queuedClient;
};

const loadMailerLiteScript = () => {
  if (document.getElementById(mailerLiteScriptID)) {
    return;
  }

  const script = document.createElement("script");
  script.id = mailerLiteScriptID;
  script.async = true;
  script.src = "https://assets.mailerlite.com/js/universal.js";
  document.head.appendChild(script);
};

export const SignupWaitlist: React.FC = () => {
  const mailerLiteConfig = getMailerLiteConfig();
  const mailerLiteAccountID = mailerLiteConfig?.accountID;
  const mailerLiteFormID = mailerLiteConfig?.formID;
  const hasMailerLiteForm = Boolean(mailerLiteFormID);

  useEffect(() => {
    if (!mailerLiteAccountID) {
      return;
    }

    const ml = ensureMailerLiteClient();
    loadMailerLiteScript();
    ml("account", mailerLiteAccountID);
  }, [mailerLiteAccountID]);

  return (
    <div className="space-y-4">
      <Text className="text-left text-sm leading-6 text-gray-600">
        We are opening access gradually while demand is high.
        {hasMailerLiteForm && " Leave your email and we will send an invite as soon as capacity is available."}
      </Text>

      {hasMailerLiteForm && <div className="ml-embedded -mx-5" data-form={mailerLiteFormID} />}
    </div>
  );
};
