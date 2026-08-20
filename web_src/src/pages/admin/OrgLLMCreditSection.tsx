import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { bpsToPercentInput, dollarInputToCents, percentInputToBps } from "@/lib/hostedCredit";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";
import { Wallet } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

type OrganizationLLMCredit = {
  remaining_credit_cents: number;
  grant_total_cents: number;
  hosted_billed_cents: number;
  markup_bps: number;
  markup_override_bps: number | null;
  warning: boolean;
};

const getErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === "") {
    return fallback;
  }

  return text;
};

export function OrgLLMCreditSection({ orgId }: { orgId: string }) {
  const [credit, setCredit] = useState<OrganizationLLMCredit | null>(null);
  const [loading, setLoading] = useState(true);
  const [grantDollars, setGrantDollars] = useState("");
  const [note, setNote] = useState("");
  const [markupPercent, setMarkupPercent] = useState("");
  const [savingGrant, setSavingGrant] = useState(false);
  const [savingMarkup, setSavingMarkup] = useState(false);

  const applyCredit = useCallback((data: OrganizationLLMCredit) => {
    setCredit(data);
    setMarkupPercent(data.markup_override_bps == null ? "" : bpsToPercentInput(data.markup_override_bps));
  }, []);

  const loadCredit = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit`, { credentials: "include" });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to load organization credit"));
      }
      applyCredit(await response.json());
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load organization credit");
    } finally {
      setLoading(false);
    }
  }, [applyCredit, orgId]);

  useEffect(() => {
    loadCredit();
  }, [loadCredit]);

  const addGrant = async () => {
    setSavingGrant(true);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit/grants`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          amount_cents: dollarInputToCents(grantDollars),
          note: note.trim(),
        }),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to add hosted credit"));
      }
      applyCredit(await response.json());
      setGrantDollars("");
      setNote("");
      showSuccessToast("Hosted credit increased");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to add hosted credit");
    } finally {
      setSavingGrant(false);
    }
  };

  const saveMarkup = async () => {
    setSavingMarkup(true);
    try {
      const body =
        markupPercent.trim() === "" ? { markup_bps: null } : { markup_bps: percentInputToBps(markupPercent) };
      const response = await fetch(`/admin/api/organizations/${orgId}/llm-settings`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error(await getErrorMessage(response, "Failed to update markup override"));
      }
      applyCredit(await response.json());
      showSuccessToast("Markup override updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update markup override");
    } finally {
      setSavingMarkup(false);
    }
  };

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Wallet size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Hosted credit
        </Heading>
      </div>

      {loading && !credit ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">Loading hosted credit...</Text>
      ) : credit ? (
        <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 p-4 dark:bg-gray-900 dark:outline-gray-700/70">
          <div className="grid gap-4 sm:grid-cols-3">
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Remaining</p>
              <p className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">
                {formatUsdCents(credit.remaining_credit_cents)}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Grant total</p>
              <p className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">
                {formatUsdCents(credit.grant_total_cents)}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Hosted billed</p>
              <p className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">
                {formatUsdCents(credit.hosted_billed_cents)}
              </p>
            </div>
          </div>
          {credit.warning ? (
            <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
              Remaining hosted credit is at or below the warning threshold.
            </Text>
          ) : null}
          {credit.remaining_credit_cents <= 0 ? (
            <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
              Hosted credit is empty. Increase credit to restore SuperPlane-hosted runs.
            </Text>
          ) : null}

          <div className="mt-6 grid gap-4 md:grid-cols-2">
            <div>
              <Label className="mb-2 block text-left">Increase credit (USD)</Label>
              <InputGroup>
                <Input
                  data-testid="admin-org-credit-amount"
                  value={grantDollars}
                  onChange={(event) => setGrantDollars(event.target.value)}
                  placeholder="10.00"
                />
              </InputGroup>
              <Label className="mb-2 mt-3 block text-left">Note (optional)</Label>
              <Textarea
                data-testid="admin-org-credit-note"
                value={note}
                onChange={(event) => setNote(event.target.value)}
                placeholder="Reason for this grant"
              />
              <Button
                type="button"
                className="mt-3"
                data-testid="admin-org-credit-grant"
                onClick={addGrant}
                disabled={savingGrant || dollarInputToCents(grantDollars) <= 0}
              >
                {savingGrant ? "Adding..." : "Increase credit"}
              </Button>
            </div>
            <div>
              <Label className="mb-2 block text-left">Markup override percent</Label>
              <InputGroup>
                <Input
                  data-testid="admin-org-markup-override"
                  value={markupPercent}
                  onChange={(event) => setMarkupPercent(event.target.value)}
                  placeholder={`Installation default (${bpsToPercentInput(credit.markup_bps)})`}
                />
              </InputGroup>
              <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Leave empty to use the installation markup. Organization members cannot see this value.
              </Text>
              <Button
                type="button"
                className="mt-3"
                data-testid="admin-org-markup-save"
                onClick={saveMarkup}
                disabled={savingMarkup}
              >
                {savingMarkup ? "Saving..." : "Save markup override"}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
