import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
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
  /** Prefills the first key name when the dialog opens (value left blank). */
  initialKeyName?: string;
}

interface KeyValuePair {
  id: string;
  name: string;
  value: string;
}

type ValidKeyValuePair = Pick<KeyValuePair, "name" | "value">;

function createEmptyPair(name = ""): KeyValuePair {
  return {
    id: `secret-key-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    name,
    value: "",
  };
}

function validateForm(name: string, pairs: KeyValuePair[]): { error: string | null; validPairs: ValidKeyValuePair[] } {
  const trimmedName = name.trim();
  if (!trimmedName) {
    return { error: "Enter a secret name.", validPairs: [] };
  }

  if (pairs.some((pair) => !pair.name.trim() || !pair.value.trim())) {
    return { error: "Enter a name and value for each key.", validPairs: [] };
  }

  const validPairs = pairs.map((pair) => ({ name: pair.name.trim(), value: pair.value }));
  const keys = validPairs.map((pair) => pair.name);
  if (new Set(keys).size !== keys.length) {
    return { error: "Use a unique name for each key.", validPairs: [] };
  }

  return { error: null, validPairs };
}

interface KeyValueRowProps {
  pair: KeyValuePair;
  index: number;
  onChange: (patch: Partial<KeyValuePair>) => void;
  onRemove: (() => void) | null;
  disabled: boolean;
}

function KeyValueRow({ pair, index, onChange, onRemove, disabled }: KeyValueRowProps) {
  const keyInputId = `${pair.id}-name`;
  const valueInputId = `${pair.id}-value`;

  return (
    <div className="grid gap-2 p-3 sm:grid-cols-[minmax(0,2fr)_minmax(0,3fr)_2rem] sm:items-start">
      <div className="min-w-0 space-y-1.5">
        <Label htmlFor={keyInputId} className="text-xs text-muted-foreground sm:sr-only">
          Key name {index + 1}
        </Label>
        <Input
          id={keyInputId}
          type="text"
          value={pair.name}
          onChange={(e) => onChange({ name: e.target.value })}
          placeholder="API_TOKEN"
          className="font-mono text-[13px]"
          disabled={disabled}
          data-testid="secrets-create-key"
        />
      </div>
      <div className="min-w-0 space-y-1.5">
        <Label htmlFor={valueInputId} className="text-xs text-muted-foreground sm:sr-only">
          Secret value {index + 1}
        </Label>
        <Textarea
          id={valueInputId}
          value={pair.value}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="Enter a value"
          rows={1}
          className="min-h-8 resize-y py-1.5 font-mono text-[13px]"
          disabled={disabled}
          data-testid="secrets-create-value"
        />
      </div>
      {onRemove && (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          onClick={onRemove}
          className="text-muted-foreground hover:text-destructive"
          aria-label={`Remove key ${index + 1}`}
          title="Remove key"
          disabled={disabled}
          data-testid="secrets-create-remove-pair"
        >
          <Trash2 className="size-3.5" aria-hidden />
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
    <section className="space-y-2" aria-labelledby="secret-keys-heading">
      <div className="flex items-end justify-between gap-4">
        <div className="space-y-1">
          <h3 id="secret-keys-heading" className="text-[13px] font-medium text-foreground">
            Secret keys
          </h3>
          <p className="text-xs text-muted-foreground">Add each key that integrations can use.</p>
        </div>
        <span className="shrink-0 text-xs text-muted-foreground">
          {pairs.length} {pairs.length === 1 ? "key" : "keys"}
        </span>
      </div>
      <div className="overflow-hidden rounded-md border border-border">
        <div className="hidden grid-cols-[minmax(0,2fr)_minmax(0,3fr)_2rem] gap-2 border-b border-border bg-muted/40 px-3 py-2 text-[11px] font-medium text-muted-foreground sm:grid">
          <span>Key name</span>
          <span>Secret value</span>
          <span className="sr-only">Actions</span>
        </div>
        <div className="max-h-[min(40vh,22rem)] divide-y divide-border overflow-y-auto">
          {pairs.map((pair, index) => (
            <KeyValueRow
              key={pair.id}
              pair={pair}
              index={index}
              onChange={(patch) => onUpdate(index, patch)}
              onRemove={pairs.length > 1 ? () => onRemove(index) : null}
              disabled={disabled}
            />
          ))}
        </div>
        <div className="border-t border-border bg-muted/20 p-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onAdd}
            disabled={disabled}
            data-testid="secrets-create-add-pair"
          >
            <Plus className="size-3.5" aria-hidden />
            Add another key
          </Button>
        </div>
      </div>
    </section>
  );
}

function initialPairs(initialKeyName?: string): KeyValuePair[] {
  const key = initialKeyName?.trim();
  return [createEmptyPair(key)];
}

function FormError({ children }: { children?: React.ReactNode }) {
  if (!children) {
    return null;
  }
  return (
    <p role="alert" className="text-xs text-destructive">
      {children}
    </p>
  );
}

export function CreateSecretDialog({
  open,
  onOpenChange,
  organizationId,
  onCreated,
  initialKeyName,
}: CreateSecretDialogProps) {
  const [secretName, setSecretName] = useState("");
  const [keyValuePairs, setKeyValuePairs] = useState<KeyValuePair[]>(() => initialPairs(initialKeyName));
  const [formError, setFormError] = useState("");
  const createSecretMutation = useCreateSecret(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const isPending = createSecretMutation.isPending;

  const reset = () => {
    setSecretName("");
    setKeyValuePairs(initialPairs(initialKeyName));
    setFormError("");
    createSecretMutation.reset();
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (isPending) return;
    onOpenChange(nextOpen);
    if (!nextOpen) reset();
  };

  const updatePair = (index: number, patch: Partial<KeyValuePair>) => {
    setFormError("");
    setKeyValuePairs((prev) => prev.map((pair, i) => (i === index ? { ...pair, ...patch } : pair)));
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const { error, validPairs } = validateForm(secretName, keyValuePairs);
    if (error) {
      setFormError(error);
      return;
    }
    try {
      const params: CreateSecretParams = { name: secretName.trim(), environmentVariables: validPairs };
      const result = await createSecretMutation.mutateAsync(params);
      showSuccessToast("Secret created.");

      const createdId = result?.data?.secret?.metadata?.id ?? "";
      const createdName = result?.data?.secret?.metadata?.name ?? params.name;

      onOpenChange(false);
      reset();
      onCreated?.({ id: createdId, name: createdName, keys: validPairs.map((p) => p.name) });
    } catch (err) {
      showErrorToast(getApiErrorMessage(err, "SuperPlane could not create the secret."));
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-hidden sm:max-w-2xl" showCloseButton={!isPending}>
        <DialogHeader>
          <DialogTitle>Create secret</DialogTitle>
          <DialogDescription>
            Store credentials for integrations. SuperPlane hides values after you save.
          </DialogDescription>
        </DialogHeader>
        <form className="flex min-h-0 flex-col gap-4" onSubmit={handleSubmit} data-testid="secrets-create-form">
          <div className="space-y-2">
            <Label htmlFor="create-secret-name">Secret name</Label>
            <Input
              id="create-secret-name"
              type="text"
              value={secretName}
              onChange={(e) => {
                setFormError("");
                setSecretName(e.target.value);
              }}
              placeholder="Production API credentials"
              autoFocus
              required
              disabled={isPending}
              data-testid="secrets-create-name"
            />
          </div>
          <KeyValuePairsSection
            pairs={keyValuePairs}
            disabled={isPending}
            onUpdate={updatePair}
            onAdd={() => {
              setFormError("");
              setKeyValuePairs((prev) => [...prev, createEmptyPair()]);
            }}
            onRemove={(index) => {
              setFormError("");
              setKeyValuePairs((prev) => prev.filter((_, i) => i !== index));
            }}
          />
          <FormError>{formError}</FormError>
          <FormError>
            {createSecretMutation.isError
              ? `SuperPlane could not create the secret. ${getApiErrorMessage(createSecretMutation.error)}`
              : undefined}
          </FormError>
          <DialogFooter>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <LoadingButton
              type="submit"
              size="sm"
              loading={isPending}
              loadingText="Creating…"
              data-testid="secrets-create-submit"
            >
              Create secret
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
