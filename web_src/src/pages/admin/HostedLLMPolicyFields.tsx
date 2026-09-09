import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

export function HostedLLMPolicyFields({
  welcomeDollars,
  welcomeTTLDays,
  markupPercent,
  warningPercent,
  savingPolicy,
  policyChanged,
  policyValid,
  onWelcomeChange,
  onWelcomeTTLChange,
  onMarkupChange,
  onWarningChange,
  onSave,
}: {
  welcomeDollars: string;
  welcomeTTLDays: string;
  markupPercent: string;
  warningPercent: string;
  savingPolicy: boolean;
  policyChanged: boolean;
  policyValid: boolean;
  onWelcomeChange: (value: string) => void;
  onWelcomeTTLChange: (value: string) => void;
  onMarkupChange: (value: string) => void;
  onWarningChange: (value: string) => void;
  onSave: () => void;
}) {
  return (
    <>
      <div className="mt-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <div>
          <Label className="mb-2 block text-left">Welcome grant (USD)</Label>
          <InputGroup>
            <Input
              data-testid="installation-llm-welcome"
              value={welcomeDollars}
              onChange={(event) => onWelcomeChange(event.target.value)}
              placeholder="50.00"
            />
          </InputGroup>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Granted once when an organization is created. Set to 0 to disable grants.
          </Text>
        </div>
        <div>
          <Label className="mb-2 block text-left">Welcome duration (days)</Label>
          <InputGroup>
            <Input
              data-testid="installation-llm-welcome-ttl"
              value={welcomeTTLDays}
              onChange={(event) => onWelcomeTTLChange(event.target.value)}
              placeholder="14"
            />
          </InputGroup>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Unused welcome credit expires after this many days. This applies to new grants.
          </Text>
        </div>
        <div>
          <Label className="mb-2 block text-left">Markup percent</Label>
          <InputGroup>
            <Input
              data-testid="installation-llm-markup"
              value={markupPercent}
              onChange={(event) => onMarkupChange(event.target.value)}
              placeholder="20"
            />
          </InputGroup>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Applied to SuperPlane-hosted spend. Organization members cannot see this value.
          </Text>
        </div>
        <div>
          <Label className="mb-2 block text-left">Warning threshold percent</Label>
          <InputGroup>
            <Input
              data-testid="installation-llm-warning"
              value={warningPercent}
              onChange={(event) => onWarningChange(event.target.value)}
              placeholder="20"
            />
          </InputGroup>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Show a warning when remaining credit is at or below this percent of the grant total.
          </Text>
        </div>
      </div>

      <div className="mt-5">
        <Button
          type="button"
          data-testid="installation-llm-policy-save"
          onClick={onSave}
          disabled={savingPolicy || !policyChanged || !policyValid}
        >
          {savingPolicy ? "Saving..." : "Save hosted LLM policy"}
        </Button>
      </div>
    </>
  );
}
