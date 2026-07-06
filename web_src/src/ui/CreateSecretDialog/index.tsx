import { useState } from "react";
import { Key, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Textarea } from "@/components/Textarea/textarea";
import { useCreateSecret, type CreateSecretParams } from "@/hooks/useSecrets";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

export interface CreatedSecretSummary {
  id: string;
  name: string;
  keys: string[];
}

export interface CreateSecretDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  /** Called after a secret is successfully created. */
  onCreated?: (secret: CreatedSecretSummary) => void;
}

interface KeyValuePair {
  name: string;
  value: string;
}

const EMPTY_PAIR: KeyValuePair = { name: "", value: "" };

function validateForm(name: string, pairs: KeyValuePair[]): { error: string | null; validPairs: KeyValuePair[] } {
  const trimmedName = name.trim();
  if (!trimmedName) return { error: "Secret name is required", validPairs: [] };

  const validPairs = pairs
    .map((p) => ({ name: p.name.trim(), value: p.value.trim() }))
    .filter((p) => p.name && p.value);
  if (validPairs.length === 0) return { error: "At least one key-value pair is required", validPairs: [] };

  const keys = validPairs.map((p) => p.name);
  if (new Set(keys).size !== keys.length) return { error: "Duplicate key names are not allowed", validPairs: [] };

  return { error: null, validPairs };
}

interface KeyValueRowProps {
  pair: KeyValuePair;
  onChange: (patch: Partial<KeyValuePair>) => void;
  onRemove: (() => void) | null;
  disabled: boolean;
}

function KeyValueRow({ pair, onChange, onRemove, disabled }: KeyValueRowProps) {
  return (
    <div className="flex gap-2 items-start">
      <div className="flex-1 min-w-0">
        <Input
          type="text"
          value={pair.name}
          onChange={(e) => onChange({ name: e.target.value })}
          placeholder="Key"
          className="mb-2"
          disabled={disabled}
          data-testid="secrets-create-key"
        />
        <Textarea
          value={pair.value}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="Value"
          rows={3}
          className="font-mono text-sm wrap-anywhere"
          disabled={disabled}
          data-testid="secrets-create-value"
        />
      </div>
      {onRemove && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRemove}
          className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 mt-0"
          title="Remove pair"
          disabled={disabled}
          data-testid="secrets-create-remove-pair"
        >
          <Trash2 className="w-3 h-3" />
        </Button>
      )}
    </div>
  );
}

interface KeyValuePairsSectionProps {
  pairs: KeyValuePair[];
  disabled: boolean;
  onUpdate: (index: number, patch: Partial<KeyValuePair>) => void;
  onAdd: () => void;
  onRemove: (index: number) => void;
}

function KeyValuePairsSection({ pairs, disabled, onUpdate, onAdd, onRemove }: KeyValuePairsSectionProps) {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
      <Label className="text-gray-800 dark:text-gray-100 mb-2 block">
        Key-Value Pairs <span className="text-red-500">*</span>
      </Label>
      <div className="space-y-3">
        {pairs.map((pair, index) => (
          <KeyValueRow
            key={index}
            pair={pair}
            onChange={(patch) => onUpdate(index, patch)}
            onRemove={pairs.length > 1 ? () => onRemove(index) : null}
            disabled={disabled}
          />
        ))}
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={onAdd}
        className="mt-3 text-xs"
        disabled={disabled}
        data-testid="secrets-create-add-pair"
      >
        <Plus className="w-3 h-3 mr-1" />
        Add Pair
      </Button>
    </div>
  );
}

export function CreateSecretDialog({ open, onOpenChange, organizationId, onCreated }: CreateSecretDialogProps) {
  const [secretName, setSecretName] = useState("");
  const [keyValuePairs, setKeyValuePairs] = useState<KeyValuePair[]>([{ ...EMPTY_PAIR }]);
  const createSecretMutation = useCreateSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const isPending = createSecretMutation.isPending;

  const reset = () => {
    setSecretName("");
    setKeyValuePairs([{ ...EMPTY_PAIR }]);
    createSecretMutation.reset();
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (isPending) return;
    onOpenChange(nextOpen);
    if (!nextOpen) reset();
  };

  const updatePair = (index: number, patch: Partial<KeyValuePair>) => {
    setKeyValuePairs((prev) => prev.map((pair, i) => (i === index ? { ...pair, ...patch } : pair)));
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const { error, validPairs } = validateForm(secretName, keyValuePairs);
    if (error) {
      showErrorToast(error);
      return;
    }
    try {
      const params: CreateSecretParams = { name: secretName.trim(), environmentVariables: validPairs };
      const result = await createSecretMutation.mutateAsync(params);
      showSuccessToast("Secret created successfully");

      const createdId = result?.data?.secret?.metadata?.id ?? "";
      const createdName = result?.data?.secret?.metadata?.name ?? params.name;

      onOpenChange(false);
      reset();
      onCreated?.({ id: createdId, name: createdName, keys: validPairs.map((p) => p.name) });
    } catch (err) {
      showErrorToast(`Failed to create secret: ${getApiErrorMessage(err)}`);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent showCloseButton={!isPending}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <Key className="w-5 h-5 text-gray-500 dark:text-gray-400" />
            Create Secret
          </DialogTitle>
        </DialogHeader>
        <form className="space-y-4" onSubmit={handleSubmit} data-testid="secrets-create-form">
          <div className="space-y-2">
            <Label htmlFor="create-secret-name" className="text-gray-800 dark:text-gray-100">
              Secret Name <span className="text-red-500">*</span>
            </Label>
            <Input
              id="create-secret-name"
              type="text"
              value={secretName}
              onChange={(e) => setSecretName(e.target.value)}
              placeholder="e.g., production-api-keys"
              required
              disabled={isPending}
              data-testid="secrets-create-name"
            />
          </div>
          <KeyValuePairsSection
            pairs={keyValuePairs}
            disabled={isPending}
            onUpdate={updatePair}
            onAdd={() => setKeyValuePairs((prev) => [...prev, { ...EMPTY_PAIR }])}
            onRemove={(index) => setKeyValuePairs((prev) => prev.filter((_, i) => i !== index))}
          />
          {createSecretMutation.isError && (
            <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
              <p className="text-sm text-red-800 dark:text-red-200">
                Failed to create secret: {getApiErrorMessage(createSecretMutation.error)}
              </p>
            </div>
          )}
          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)} disabled={isPending}>
              Cancel
            </Button>
            <LoadingButton
              type="submit"
              color="blue"
              disabled={!secretName.trim()}
              loading={isPending}
              loadingText="Creating..."
              className="flex items-center gap-2"
              data-testid="secrets-create-submit"
            >
              Create Secret
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
