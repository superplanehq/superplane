import React from "react";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/ui/switch";
import { ErrorBanner } from "./ErrorBanner";

type PrivateNetworkStepProps = {
  allowPrivateNetworkAccess: boolean;
  loading: boolean;
  error: string | null;
  onAllowPrivateNetworkAccessChange: (checked: boolean) => void;
  onBack: () => void;
  onNext: () => void;
};

export const PrivateNetworkStep: React.FC<PrivateNetworkStepProps> = ({
  allowPrivateNetworkAccess,
  loading,
  error,
  onAllowPrivateNetworkAccessChange,
  onBack,
  onNext,
}) => (
  <div className="space-y-6">
    <ErrorBanner message={error} />

    <div className="text-left">
      <h4 className="mb-2 text-lg font-medium text-gray-800 dark:text-white">Private network access</h4>
      <Text className="text-gray-800 dark:text-gray-300">
        Decide whether this SuperPlane instance should be allowed to reach internal services before continuing to email
        setup.
      </Text>
    </div>

    <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-4 text-left dark:border-gray-700/70 dark:bg-gray-800/60">
      <div className="flex items-start justify-between gap-6">
        <div className="max-w-xs">
          <Label className="mb-1 block text-sm font-medium">Allow private network targets</Label>
          <Text className="text-sm text-gray-600 dark:text-gray-400">
            Enable this if SuperPlane needs to reach tools inside your VPC, private Kubernetes cluster, or another
            closed network. This reduces SSRF protection for private addresses.
          </Text>
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {allowPrivateNetworkAccess ? "Enabled" : "Disabled"}
          </span>
          <Switch
            data-testid="owner-setup-private-network-switch"
            checked={allowPrivateNetworkAccess}
            onCheckedChange={onAllowPrivateNetworkAccessChange}
            aria-label="Allow connections to private network tools"
          />
        </div>
      </div>
    </div>

    <div className="flex flex-wrap items-center gap-3">
      <Button type="button" variant="outline" disabled={loading} onClick={onBack}>
        Back
      </Button>
      <Button type="button" className="ml-auto" disabled={loading} onClick={onNext}>
        Next
      </Button>
    </div>
  </div>
);
