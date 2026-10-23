import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { bpsToPercentInput, centsToDollarInput, parseDollarInputToCents } from "@/lib/hostedCredit";
import { showSuccessToast } from "@/lib/toast";
import { CreditPacks } from "./CreditPacks";
import {
  fetchInstallationLLMSettings,
  patchInstallationLLMPolicy,
  type InstallationLLMSettings,
} from "./hostedLLMSettingsApi";

function parseFeeInput(value: string): number | null {
  if (!/^\d+(\.\d{1,2})?$/.test(value.trim())) return null;
  const amount = parseDollarInputToCents(value);
  return amount !== null && Number.isSafeInteger(amount) ? amount : null;
}

function useCreditPurchaseFees() {
  const [settings, setSettings] = useState<InstallationLLMSettings | null>(null);
  const [percent, setPercent] = useState("");
  const [fixedUSD, setFixedUSD] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const applySettings = useCallback((next: InstallationLLMSettings) => {
    setSettings(next);
    setPercent(bpsToPercentInput(next.credit_purchase_fee_bps));
    setFixedUSD(centsToDollarInput(next.credit_purchase_fixed_fee_cents));
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      applySettings(await fetchInstallationLLMSettings());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not load fee settings. Try again.");
    } finally {
      setLoading(false);
    }
  }, [applySettings]);

  useEffect(() => {
    void load();
  }, [load]);
  useReportPageReady(!loading);

  const bps = parseFeeInput(percent);
  const fixedCents = parseFeeInput(fixedUSD);
  const percentValid = bps !== null && bps < 10000;
  const valid = percentValid && fixedCents !== null;
  const changed =
    settings !== null &&
    (bps !== settings.credit_purchase_fee_bps || fixedCents !== settings.credit_purchase_fixed_fee_cents);

  const save = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!valid || bps === null || fixedCents === null || saving) return;
    setSaving(true);
    setError(null);
    try {
      applySettings(
        await patchInstallationLLMPolicy({
          credit_purchase_fee_bps: bps,
          credit_purchase_fixed_fee_cents: fixedCents,
        }),
      );
      showSuccessToast("Fee settings saved");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not save fee settings. Try again.");
    } finally {
      setSaving(false);
    }
  };

  return {
    settings,
    percent,
    setPercent,
    fixedUSD,
    setFixedUSD,
    loading,
    saving,
    error,
    load,
    percentValid,
    fixedValid: fixedCents !== null,
    canSave: valid && changed && !saving,
    save,
  };
}

export function Credits() {
  const {
    settings,
    percent,
    setPercent,
    fixedUSD,
    setFixedUSD,
    loading,
    saving,
    error,
    load,
    percentValid,
    fixedValid,
    canSave,
    save,
  } = useCreditPurchaseFees();

  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Credits</h1>
      <Text className="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
        Manage credit packs and processing fees for hosted credit purchases across all organizations.
      </Text>
      <CreditPacks />
      {error ? (
        <p role="alert" className="mt-5 text-sm text-red-700 dark:text-red-400">
          {error}
        </p>
      ) : null}
      {loading ? <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading fee settings...</Text> : null}
      {!loading && !settings ? (
        <Button className="mt-4" variant="outline" onClick={() => void load()}>
          Retry
        </Button>
      ) : null}
      {settings ? (
        <form onSubmit={save} className="mt-5 max-w-2xl space-y-6 bg-white px-6 py-6 dark:bg-gray-900">
          <div>
            <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100">Credit purchase fees</h2>
            <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
              SuperPlane calculates the purchase subtotal to cover these fees and shows it before Polar checkout. Set
              both values to zero to charge no processing fee.
            </Text>
          </div>
          <div className="grid gap-5 sm:grid-cols-2">
            <div>
              <Label htmlFor="credit-purchase-fee-percent" className="mb-2 block">
                Processing fee percent
              </Label>
              <Input
                id="credit-purchase-fee-percent"
                inputMode="decimal"
                value={percent}
                disabled={saving}
                onChange={(event) => setPercent(event.target.value)}
                aria-invalid={!percentValid}
                aria-describedby={!percentValid ? "credit-purchase-fee-percent-error" : undefined}
              />
              {!percentValid ? (
                <p id="credit-purchase-fee-percent-error" className="mt-1 text-xs text-red-700 dark:text-red-400">
                  Enter a percent from 0 to 99.99, with up to 2 decimal places.
                </p>
              ) : null}
            </div>
            <div>
              <Label htmlFor="credit-purchase-fixed-fee" className="mb-2 block">
                Fixed fee per purchase (USD)
              </Label>
              <Input
                id="credit-purchase-fixed-fee"
                inputMode="decimal"
                value={fixedUSD}
                disabled={saving}
                onChange={(event) => setFixedUSD(event.target.value)}
                aria-invalid={!fixedValid}
                aria-describedby={!fixedValid ? "credit-purchase-fixed-fee-error" : undefined}
              />
              {!fixedValid ? (
                <p id="credit-purchase-fixed-fee-error" className="mt-1 text-xs text-red-700 dark:text-red-400">
                  Enter a valid USD amount of zero or more, with up to 2 decimal places.
                </p>
              ) : null}
            </div>
          </div>
          <Button type="submit" disabled={!canSave}>
            {saving ? "Saving..." : "Save fee settings"}
          </Button>
        </form>
      ) : null}
    </div>
  );
}
