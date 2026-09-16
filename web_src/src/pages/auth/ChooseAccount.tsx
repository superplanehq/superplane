import React, { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";

import superplaneLogo from "@/assets/superplane.svg";
import { Button } from "@/components/ui/button";
import { useReportPageReady } from "@/hooks/useReportPageReady";

type AccountChoice = {
  id: string;
  name: string;
  email: string;
  avatar_url?: string;
};

const loadErrorMessage = "SuperPlane could not load the accounts. Sign in again.";
const submitErrorMessage = "SuperPlane could not open that account. Try again.";
const missingTokenMessage = "This page needs a valid account selection link. Sign in again.";

async function loadAccountChoices(token: string): Promise<AccountChoice[]> {
  const response = await fetch(`/auth/choose-account?token=${encodeURIComponent(token)}`, {
    headers: { Accept: "application/json" },
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(loadErrorMessage);
  }
  const data = (await response.json()) as { accounts?: AccountChoice[] };
  return data.accounts ?? [];
}

async function submitAccountChoice(token: string, accountId: string): Promise<string> {
  const body = new URLSearchParams();
  body.set("token", token);
  body.set("account_id", accountId);
  const response = await fetch("/auth/choose-account", {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/x-www-form-urlencoded",
    },
    credentials: "include",
    body: body.toString(),
  });
  if (!response.ok) {
    throw new Error(submitErrorMessage);
  }
  const data = (await response.json()) as { redirectUrl?: string };
  return data.redirectUrl || "/";
}

export const ChooseAccount: React.FC = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token")?.trim() ?? "";
  const [accounts, setAccounts] = useState<AccountChoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(token ? null : missingTokenMessage);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    let canceled = false;
    void loadAccountChoices(token)
      .then((items) => {
        if (canceled) {
          return;
        }
        setAccounts(items);
        if (items.length === 0) {
          setError(loadErrorMessage);
        }
      })
      .catch(() => {
        if (!canceled) {
          setError(loadErrorMessage);
        }
      })
      .finally(() => {
        if (!canceled) {
          setLoading(false);
        }
      });

    return () => {
      canceled = true;
    };
  }, [token]);

  useReportPageReady(!loading, { failed: !!error });

  const chooseAccount = async (accountId: string) => {
    setBusy(true);
    setError(null);
    try {
      window.location.href = await submitAccountChoice(token, accountId);
    } catch {
      setError(submitErrorMessage);
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col bg-gray-400 px-4 py-10 dark:bg-gray-950">
      <div className="flex flex-1 flex-col items-center justify-center">
        <div className="w-full max-w-sm rounded-3xl bg-white p-8 shadow-sm outline outline-gray-950/10 dark:bg-gray-900 dark:outline-gray-700/70">
          <div className="text-center">
            <img src={superplaneLogo} alt="SuperPlane logo" className="mx-auto h-8 w-8 dark:brightness-0 dark:invert" />
            <h1 className="mt-2 !text-lg font-medium text-gray-900 dark:text-gray-100">Choose an account</h1>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
              This GitHub identity can sign in to more than one SuperPlane account.
            </p>
          </div>
          <AccountChoiceBody
            loading={loading}
            error={error}
            accounts={accounts}
            busy={busy}
            onChoose={(accountId) => {
              void chooseAccount(accountId);
            }}
          />
        </div>
      </div>
    </div>
  );
};

function AccountChoiceBody({
  loading,
  error,
  accounts,
  busy,
  onChoose,
}: {
  loading: boolean;
  error: string | null;
  accounts: AccountChoice[];
  busy: boolean;
  onChoose: (accountId: string) => void;
}) {
  return (
    <div className="pt-6">
      {loading && <p className="text-sm text-gray-500 dark:text-gray-400">Loading accounts...</p>}
      {error && (
        <div className="mb-4 rounded-md border border-red-300 bg-white px-3 py-1 text-sm text-red-500 dark:border-red-500/40 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </div>
      )}
      {!loading && accounts.length > 0 && (
        <ul className="space-y-3">
          {accounts.map((account) => (
            <li key={account.id}>
              <AccountChoiceButton account={account} disabled={busy} onChoose={onChoose} />
            </li>
          ))}
        </ul>
      )}
      <div className="mt-6 text-sm text-gray-500 dark:text-gray-400">
        <Link to="/login" className="font-medium text-gray-900 underline underline-offset-2 dark:text-gray-100">
          Back to sign in
        </Link>
      </div>
    </div>
  );
}

function AccountChoiceButton({
  account,
  disabled,
  onChoose,
}: {
  account: AccountChoice;
  disabled: boolean;
  onChoose: (accountId: string) => void;
}) {
  const initial = (account.name || account.email || "?").slice(0, 1).toUpperCase();
  return (
    <Button
      type="button"
      variant="outline"
      className="h-auto w-full justify-start gap-3 py-3"
      disabled={disabled}
      onClick={() => onChoose(account.id)}
    >
      {account.avatar_url ? (
        <img src={account.avatar_url} alt="" className="h-8 w-8 rounded-full" />
      ) : (
        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm text-gray-700 dark:bg-gray-700 dark:text-gray-200">
          {initial}
        </span>
      )}
      <span className="min-w-0 text-left">
        <span className="block truncate font-medium">{account.name}</span>
        <span className="block truncate text-xs font-normal text-gray-500 dark:text-gray-400">{account.email}</span>
      </span>
    </Button>
  );
}
