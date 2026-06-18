import type React from "react";
import { useEffect } from "react";
import { Text } from "@/components/Text/text";

const mailerLiteScriptID = "mailerlite-universal-script";
const mailerLiteAccountID = "1905484";

type MailerLiteClient = {
  (...args: unknown[]): void;
  q?: unknown[][];
};

type MailerLiteWindow = Window & {
  ml?: MailerLiteClient;
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
  useEffect(() => {
    const ml = ensureMailerLiteClient();
    loadMailerLiteScript();
    ml("account", mailerLiteAccountID);
  }, []);

  return (
    <div className="space-y-4">
      <div className="space-y-2 text-left">
        <h2 className="text-base font-semibold text-gray-900">Join the SuperPlane Cloud waitlist</h2>
        <Text className="text-sm leading-6 text-gray-600">
          We are opening access gradually while demand is high. Leave your email and we will send an invite as soon as
          capacity is available.
        </Text>
      </div>

      <div className="ml-embedded -mx-5" data-form="BwOnd2" />
    </div>
  );
};
