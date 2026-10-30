import React, { useCallback, useEffect, useState } from "react";
import { posthog } from "@/posthog";
import { consumePendingSignupAnalyticsPreference, hasConfirmedSignupAnalyticsPreference } from "@/lib/signupAnalytics";
import { getUtmEventProperties } from "@/lib/utmAttribution";
import { trackGoogleSignup } from "@/lib/googleTagManager";
import { setSupportSessionActive } from "@/lib/supportSession";

import { AccountContext, type AccountContextType } from "./accountContextState";

interface AccountProviderProps {
  children: React.ReactNode;
}

export function AccountProvider({ children }: AccountProviderProps) {
  const [account, setAccount] = useState<AccountContextType["account"]>(null);
  const [loading, setLoading] = useState(true);
  const [setupRequired, setSetupRequired] = useState(false);

  const loadAccount = useCallback(async (identify: boolean, keepAccountOnError = false) => {
    const response = await fetch("/account", {
      method: "GET",
      credentials: "include",
      redirect: "manual",
    });

    if (response.status === 409 && response.headers.get("X-Owner-Setup-Required") === "true") {
      setSetupRequired(true);
      setAccount(null);
      return;
    }

    if (response.status !== 200) {
      if (!keepAccountOnError || response.status === 401) {
        setSupportSessionActive(false);
        setAccount(null);
      }
      return;
    }

    const accountData = await response.json();
    const inSupportSession = accountData.support_session?.active === true;
    setSupportSessionActive(inSupportSession);
    setAccount(accountData);

    if (!identify || inSupportSession) {
      return;
    }

    identifySignedInAccount(accountData);
  }, []);

  useEffect(() => {
    loadAccount(true)
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, [loadAccount]);

  const refreshAccount = useCallback(async () => {
    await loadAccount(false, true);
  }, [loadAccount]);

  return (
    <AccountContext.Provider value={{ account, loading, setupRequired, refreshAccount }}>
      {children}
    </AccountContext.Provider>
  );
}

interface SignedInAccount {
  id: string;
  email: string;
  name: string;
  installation_admin: boolean;
}

function identifySignedInAccount(accountData: SignedInAccount) {
  const signupResult = getSignupAnalyticsResult(window.location.search);
  if (
    signupResult === "created" ||
    (signupResult !== "existing" && hasConfirmedSignupAnalyticsPreference(accountData.email))
  ) {
    trackGoogleSignup(accountData.id);
  }
  const signupPreference = consumePendingSignupAnalyticsPreference({
    accountEmail: accountData.email,
    currentPath: window.location.pathname,
    signupResult,
  });

  const accountProperties = {
    email: accountData.email,
    name: accountData.name,
    installation_admin: accountData.installation_admin,
    ...(signupPreference
      ? {
          product_updates_opt_in: signupPreference.productUpdatesOptIn,
        }
      : {}),
  };

  posthog.identify(accountData.id, accountProperties);

  if (signupPreference || signupResult === "created") {
    const signupProperties: Record<string, unknown> = {
      ...getUtmEventProperties(),
    };
    if (signupPreference) {
      signupProperties.product_updates_opt_in = signupPreference.productUpdatesOptIn;
      signupProperties.$set = {
        product_updates_opt_in: signupPreference.productUpdatesOptIn,
      };
    }
    if (Object.keys(signupProperties).length === 0) {
      posthog.capture("user_signed_up");
    } else {
      posthog.capture("user_signed_up", signupProperties);
    }
  }

  if (signupResult) {
    removeSignupAnalyticsResult();
  }
}

function getSignupAnalyticsResult(search: string) {
  const value = new URLSearchParams(search).get("auth_signup_result");
  if (value === "created" || value === "existing") {
    return value;
  }

  return null;
}

function removeSignupAnalyticsResult() {
  const params = new URLSearchParams(window.location.search);
  params.delete("auth_signup_result");

  const search = params.toString();
  const nextURL = `${window.location.pathname}${search ? `?${search}` : ""}${window.location.hash}`;
  window.history.replaceState(window.history.state, "", nextURL);
}
