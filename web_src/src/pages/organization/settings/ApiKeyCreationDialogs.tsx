import { Icon } from "@/components/Icon";
import { Textarea } from "@/components/Textarea/textarea";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { useCreateAPIKey } from "@/hooks/useApiKeys";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import { KeyRound } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";
import { settingsModalClassName } from "./settingsPageStyles";

type AccessMode = "organization" | "canvas";

interface CreateApiKeyFormState {
  isCreateModalOpen: boolean;
  name: string;
  setName: Dispatch<SetStateAction<string>>;
  description: string;
  setDescription: Dispatch<SetStateAction<string>>;
  role: string;
  setRole: Dispatch<SetStateAction<string>>;
  expiresAt: string;
  setExpiresAt: Dispatch<SetStateAction<string>>;
  accessMode: AccessMode;
  setAccessMode: Dispatch<SetStateAction<AccessMode>>;
  selectedCanvasIds: string[];
  newToken: string | null;
  createMutation: ReturnType<typeof useCreateAPIKey>;
  handleCloseCreateModal: () => void;
  handleCreate: () => Promise<void>;
  handleTokenModalClose: () => void;
  toggleCanvas: (canvasId: string) => void;
}

interface RoleQueryStatusProps {
  isLoading: boolean;
  isFetching: boolean;
  hasError: boolean;
  onRetry: () => void;
}

interface CreateApiKeyFieldsProps {
  form: CreateApiKeyFormState;
  canvases: ReadonlyArray<{ id?: string; name?: string }>;
  assignableRoles: ReadonlyArray<{ name: string; label: string }>;
  roleQueryStatus: RoleQueryStatusProps;
}

function RoleQueryStatus({ isLoading, isFetching, hasError, onRetry }: RoleQueryStatusProps) {
  if (isLoading) {
    return (
      <p role="status" aria-live="polite" className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        Loading custom roles...
      </p>
    );
  }

  if (hasError) {
    return (
      <div role="alert" className="mt-1 flex items-center gap-2 text-xs text-amber-700 dark:text-amber-300">
        <span>Custom roles could not be loaded. Available options may be incomplete.</span>
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto p-0 text-xs text-current"
          onClick={onRetry}
          disabled={isFetching}
        >
          {isFetching ? "Retrying..." : "Retry"}
        </Button>
      </div>
    );
  }

  if (isFetching) {
    return (
      <p role="status" aria-live="polite" className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        Refreshing custom roles...
      </p>
    );
  }

  return null;
}

function CreateApiKeyFields({ form, canvases, assignableRoles, roleQueryStatus }: CreateApiKeyFieldsProps) {
  return (
    <div className="space-y-4">
      <div>
        <Label className="text-gray-800 dark:text-gray-100 mb-2">
          Name <span className="text-red-500">*</span>
        </Label>
        <Input
          type="text"
          value={form.name}
          onChange={(event) => form.setName(event.target.value)}
          placeholder="e.g., ci-deploy-bot"
          required
          data-testid="api-key-create-name"
        />
      </div>
      <div>
        <Label className="text-gray-800 dark:text-gray-100 mb-2">Description</Label>
        <Textarea
          value={form.description}
          onChange={(event) => form.setDescription(event.target.value)}
          placeholder="What is this API key used for?"
          rows={3}
          data-testid="api-key-create-description"
        />
      </div>
      <div>
        <Label className="text-gray-800 dark:text-gray-100 mb-2">
          Role <span className="text-red-500">*</span>
        </Label>
        <Select value={form.role} onValueChange={form.setRole}>
          <SelectTrigger className="w-full" data-testid="api-key-create-role">
            <SelectValue placeholder="Select a role" />
          </SelectTrigger>
          <SelectContent>
            {assignableRoles.map((role) => (
              <SelectItem key={role.name} value={role.name}>
                {role.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          Determines what this API key can do within its access scope.
        </p>
        <RoleQueryStatus {...roleQueryStatus} />
      </div>
      <div>
        <Label className="text-gray-800 dark:text-gray-100 mb-2">Access</Label>
        <Select value={form.accessMode} onValueChange={(value) => form.setAccessMode(value as AccessMode)}>
          <SelectTrigger className="w-full" data-testid="api-key-create-access-mode">
            <SelectValue placeholder="Select access" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="organization">Organization-wide</SelectItem>
            <SelectItem value="canvas">Selected apps</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {form.accessMode === "canvas" && (
        <div>
          <Label className="text-gray-800 dark:text-gray-100 mb-2">
            Apps <span className="text-red-500">*</span>
          </Label>
          <div className="max-h-44 overflow-y-auto rounded-md border border-gray-200 dark:border-gray-700">
            {canvases.map((canvas) => {
              const canvasId = canvas.id || "";
              return (
                <label
                  key={canvasId}
                  className="flex items-center gap-2 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 dark:border-gray-800"
                >
                  <Checkbox
                    checked={form.selectedCanvasIds.includes(canvasId)}
                    onChange={() => form.toggleCanvas(canvasId)}
                    data-testid="api-key-create-canvas"
                  />
                  <span className="text-gray-800 dark:text-gray-100">{canvas.name || "Unnamed"}</span>
                </label>
              );
            })}
          </div>
        </div>
      )}
      <div>
        <Label className="text-gray-800 dark:text-gray-100 mb-2">Expiration</Label>
        <Input
          type="datetime-local"
          value={form.expiresAt}
          onChange={(event) => form.setExpiresAt(event.target.value)}
          data-testid="api-key-create-expires-at"
        />
      </div>
    </div>
  );
}

export function CreateApiKeyModal(props: CreateApiKeyFieldsProps) {
  const { form } = props;
  if (!form.isCreateModalOpen || form.newToken) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className={cn(settingsModalClassName, "max-w-lg")}>
        <form
          className="p-6"
          onSubmit={(event) => {
            event.preventDefault();
            form.handleCreate();
          }}
          data-testid="api-key-create-form"
        >
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <KeyRound className="w-6 h-6 text-gray-500 dark:text-gray-400" />
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Create API Key</h3>
            </div>
            <button
              type="button"
              onClick={form.handleCloseCreateModal}
              className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
              disabled={form.createMutation.isPending}
            >
              <Icon name="x" size="sm" />
            </button>
          </div>

          <CreateApiKeyFields {...props} />

          <div className="flex justify-start gap-3 mt-6">
            <LoadingButton
              type="submit"
              disabled={!form.name?.trim()}
              loading={form.createMutation.isPending}
              loadingText="Creating..."
              className="flex items-center gap-2"
              data-testid="api-key-create-submit"
            >
              Create
            </LoadingButton>
            <Button
              type="button"
              variant="outline"
              onClick={form.handleCloseCreateModal}
              disabled={form.createMutation.isPending}
            >
              Cancel
            </Button>
          </div>

          {form.createMutation.isError && (
            <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
              <p className="text-sm text-red-800 dark:text-red-200">
                Failed to create: {getApiErrorMessage(form.createMutation.error)}
              </p>
            </div>
          )}
        </form>
      </div>
    </div>
  );
}

export function CreatedApiKeyModal({ form }: { form: CreateApiKeyFormState }) {
  if (!form.isCreateModalOpen || !form.newToken) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className={cn(settingsModalClassName, "max-w-lg")}>
        <div className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <KeyRound className="h-6 w-6 text-green-600 dark:text-green-400" />
            <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">API Key Created</h3>
          </div>

          <div className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md mb-4">
            <p className="text-sm text-amber-800 dark:text-amber-200">
              Copy this token now. You won't be able to see it again.
            </p>
          </div>

          <div className="flex items-center gap-2 ph-no-capture">
            <Input
              readOnly
              value={form.newToken}
              className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-800"
              data-testid="api-key-token-display"
            />
            <CopyButton
              variant="button"
              text={form.newToken}
              data-testid="api-key-token-copy"
              onCopyError={() => showErrorToast("Failed to copy token")}
            >
              Copy
            </CopyButton>
          </div>

          <div className="flex justify-start mt-6">
            <Button onClick={form.handleTokenModalClose} data-testid="api-key-token-done">
              Done
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
