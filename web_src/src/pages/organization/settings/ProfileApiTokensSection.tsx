import { Heading } from "@/components/Heading/heading";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import type { RevealedPersonalToken } from "@/hooks/usePersonalTokensPanel";
import { CopyButton } from "@/ui/CopyButton";
import { showErrorToast } from "@/lib/toast.ts";
import type { MeUserApiToken } from "@/api-client/types.gen";
import { settingsCardClassName } from "./settingsPageStyles";

function formatTokenDate(value?: string) {
  if (!value) return "Never";
  return new Date(value).toLocaleString();
}

interface ProfileApiTokensSectionProps {
  organizationId: string | null | undefined;
}

/**
 * Named personal API token management for the Profile page: create a
 * token, reveal its secret once, and list existing tokens with a per-row
 * revoke action. Mirrors the org API keys list UX in ApiKeysContent.tsx.
 */
export function ProfileApiTokensSection({ organizationId }: ProfileApiTokensSectionProps) {
  const panel = usePersonalTokensPanel(organizationId);

  return (
    <>
      <Heading level={2} className="text-lg text-left font-medium text-gray-800 dark:text-white mb-0">
        API Tokens
      </Heading>
      <Text className="text-gray-800 text-left dark:text-gray-400 text-sm">
        Use a personal API token to authenticate API requests to SuperPlane. Keep your tokens secure and do not share
        them.
      </Text>

      <div className={settingsCardClassName}>
        <div className="space-y-4">
          {panel.actionError && <Text className="text-sm text-red-500">{panel.actionError}</Text>}

          <CreateTokenForm
            name={panel.newTokenName}
            onNameChange={panel.setNewTokenName}
            onSubmit={panel.createToken}
            submitting={panel.isCreating}
          />

          {panel.revealedToken && (
            <RevealedTokenPanel
              revealedToken={panel.revealedToken}
              tokenVisible={panel.tokenVisible}
              onToggleTokenVisible={() => panel.setTokenVisible(!panel.tokenVisible)}
            />
          )}

          {!panel.tokensLoading && panel.tokens.length === 0 && (
            <div className="flex items-center gap-2">
              <Icon name="key-round" className="text-gray-500 dark:text-gray-400 text-lg" />
              <Text className="text-sm font-medium text-gray-500 dark:text-gray-400">No API tokens yet</Text>
            </div>
          )}

          {panel.tokens.length > 0 && (
            <TokenList tokens={panel.tokens} revokingId={panel.revokingId} onRevoke={panel.revokeToken} />
          )}
        </div>
      </div>
    </>
  );
}

function CreateTokenForm({
  name,
  onNameChange,
  onSubmit,
  submitting,
}: {
  name: string;
  onNameChange: (value: string) => void;
  onSubmit: () => Promise<void>;
  submitting: boolean;
}) {
  return (
    <form
      className="flex items-end gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        void onSubmit();
      }}
    >
      <div className="flex-1 max-w-xs">
        <label htmlFor="new-token-name" className="block text-sm font-medium text-gray-800 dark:text-gray-300 mb-1">
          Token name
        </label>
        <Input
          id="new-token-name"
          type="text"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="e.g., CI token"
          data-testid="user-token-create-name"
        />
      </div>
      <LoadingButton
        type="submit"
        disabled={!name.trim()}
        loading={submitting}
        loadingText="Creating..."
        className="flex items-center gap-2"
        data-testid="user-token-create-submit"
      >
        <Icon name="plus" />
        Create token
      </LoadingButton>
    </form>
  );
}

function RevealedTokenPanel({
  revealedToken,
  tokenVisible,
  onToggleTokenVisible,
}: {
  revealedToken: RevealedPersonalToken;
  tokenVisible: boolean;
  onToggleTokenVisible: () => void;
}) {
  return (
    <div className="space-y-3">
      <Text className="text-sm font-medium text-gray-700 dark:text-gray-300">New API token</Text>
      <div className="flex items-center gap-2 ph-no-capture">
        <Input
          type={tokenVisible ? "text" : "password"}
          value={revealedToken.plaintext}
          readOnly
          className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-900"
          data-testid="user-token-reveal-value"
        />
        <Button
          variant="outline"
          onClick={onToggleTokenVisible}
          className="flex items-center gap-1"
          aria-label="Toggle token visibility"
        >
          <Icon name={tokenVisible ? "eye-closed" : "eye"} />
        </Button>
        <CopyButton
          variant="button"
          text={revealedToken.plaintext}
          onCopyError={() => showErrorToast("Failed to copy API token.")}
          data-testid="user-token-reveal-copy"
        >
          Copy
        </CopyButton>
      </div>
      <div className="bg-orange-50 dark:bg-amber-900/20 border border-amber-950/15 dark:border-amber-100/15 rounded-lg p-3">
        <div className="flex items-start gap-2">
          <Icon name="key-round" className="text-amber-800 dark:text-amber-400 text-sm mt-0.5" />
          <Text className="text-amber-800 dark:text-amber-200 text-sm">
            <strong>Important:</strong> This token is shown once. Copy and store it securely. If you lose it, revoke it
            and create a new one.
          </Text>
        </div>
      </div>
    </div>
  );
}

function TokenList({
  tokens,
  revokingId,
  onRevoke,
}: {
  tokens: MeUserApiToken[];
  revokingId: string | null;
  onRevoke: (id: string, name: string) => void;
}) {
  return (
    <div className="divide-y divide-gray-100 dark:divide-gray-800" data-testid="user-token-list">
      {tokens.map((tokenItem) => (
        <div key={tokenItem.id} className="flex items-center justify-between gap-4 py-3" data-testid="user-token-row">
          <div className="min-w-0">
            <Text className="text-sm font-medium text-gray-800 dark:text-gray-100 truncate">
              {tokenItem.name || "Unnamed"}
            </Text>
            <Text className="text-xs text-gray-500 dark:text-gray-400">
              Created {formatTokenDate(tokenItem.createdAt)} · Last used {formatTokenDate(tokenItem.lastUsedAt)}
            </Text>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onRevoke(tokenItem.id || "", tokenItem.name || "")}
            disabled={revokingId === tokenItem.id}
            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 shrink-0"
            data-testid="user-token-revoke-btn"
          >
            <Icon name="trash-2" size="sm" />
            Revoke
          </Button>
        </div>
      ))}
    </div>
  );
}
