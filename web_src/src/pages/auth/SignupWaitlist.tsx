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
    <div className="space-y-5">
      <div className="space-y-2 text-center">
        <h2 className="text-lg font-semibold text-gray-900">SuperPlane Cloud is opening in waves</h2>
        <Text className="text-sm text-gray-600">
          We are managing access while traffic is high. Join the waitlist and we will email you as capacity opens.
        </Text>
      </div>

      <div className="ml-embedded -mx-5" data-form="BwOnd2" />
    </div>
  );
};
