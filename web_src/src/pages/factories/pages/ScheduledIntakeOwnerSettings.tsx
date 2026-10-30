import { useState } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";

export function ScheduledIntakeOwnerSettings({
  organizationId,
  factoryId,
  node,
  onSave,
  initialOwnerId,
  onSaved,
}: {
  organizationId: string;
  factoryId: string;
  node: SuperplaneComponentsNode;
  onSave: (update: NodeConfigurationSave) => Promise<void>;
  initialOwnerId?: string;
  onSaved?: () => void;
}) {
  const layout = useOptionalFactoriesLayout();
  const { canAct } = usePermissions();
  const [value, setValue] = useState(initialOwnerId ?? String(node.configuration?.defaultOwnerId ?? ""));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  const required = layout?.factory?.onboarding?.codexSubscription === true;
  const canUpdate = canAct("factories", "update");
  return (
    <div className="space-y-4 p-6">
      <IntakeDefaultOwnerField
        organizationId={organizationId}
        factoryId={factoryId}
        required={required}
        value={value}
        onChange={setValue}
        disabled={pending || !canUpdate}
      />
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
      <Button
        disabled={(required && !value) || pending || !canUpdate}
        onClick={async () => {
          setError(undefined);
          setPending(true);
          try {
            await onSave({
              nodeId: node.id!,
              name: node.name ?? "Create task",
              configuration: { ...node.configuration, defaultOwnerId: value },
              integration: node.integration,
              concurrency: node.concurrency,
            });
            onSaved?.();
          } catch (cause) {
            setError(getApiErrorMessage(cause, "SuperPlane could not save the default owner. Try again."));
          } finally {
            setPending(false);
          }
        }}
      >
        {pending ? "Saving…" : "Save default owner"}
      </Button>
    </div>
  );
}
