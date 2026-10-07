import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import type { PersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { showErrorToast } from "@/lib/toast";
import { CopyButton } from "@/ui/CopyButton";

/**
 * The dialogs of the personal API token flow: name a token, copy the secret
 * once, and confirm a revoke. Render this next to the token table on any page
 * that uses `usePersonalTokensPanel`.
 */
export function PersonalApiTokenDialogs({ panel }: { panel: PersonalTokensPanel }) {
  return (
    <>
      <CreateTokenDialog panel={panel} />
      <TokenSecretDialog panel={panel} />
      <RevokeTokenDialog panel={panel} />
    </>
  );
}

function CreateTokenDialog({ panel }: { panel: PersonalTokensPanel }) {
  return (
    <Dialog
      open={panel.isCreateOpen}
      onOpenChange={(open) => {
        if (!open) panel.closeCreateDialog();
      }}
    >
      <DialogContent showCloseButton={!panel.isCreating}>
        <DialogHeader>
          <DialogTitle>Create API token</DialogTitle>
          <DialogDescription>Name the token so that you can recognize where you use it.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            void panel.createToken();
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="personal-token-name">Token name</Label>
            <Input
              id="personal-token-name"
              autoFocus
              value={panel.newTokenName}
              onChange={(event) => panel.setNewTokenName(event.target.value)}
              placeholder="Deploy script"
              disabled={panel.isCreating}
              data-testid="user-token-create-name"
            />
          </div>
          {panel.createError && (
            <p className="text-sm text-red-600 dark:text-red-400" role="alert">
              {panel.createError}
            </p>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={panel.closeCreateDialog} disabled={panel.isCreating}>
              Cancel
            </Button>
            <LoadingButton
              type="submit"
              disabled={!panel.newTokenName.trim()}
              loading={panel.isCreating}
              loadingText="Creating..."
              data-testid="user-token-create-submit"
            >
              Create token
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function TokenSecretDialog({ panel }: { panel: PersonalTokensPanel }) {
  const revealed = panel.revealedToken;

  return (
    <Dialog
      open={!!revealed}
      onOpenChange={(open) => {
        if (!open) panel.dismissRevealedToken();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Copy your API token</DialogTitle>
          <DialogDescription>
            SuperPlane shows this token one time only. Copy it now and keep it in a safe place.
          </DialogDescription>
        </DialogHeader>
        <div className="ph-no-capture flex items-start gap-2 rounded-md border border-border bg-muted/40 p-3">
          <code
            className="flex-1 break-all font-mono text-xs leading-5 text-foreground"
            data-testid="user-token-reveal-value"
          >
            {revealed?.plaintext}
          </code>
          <CopyButton
            variant="button"
            text={revealed?.plaintext || ""}
            onCopyError={() => showErrorToast("Failed to copy the API token.")}
            className="shrink-0"
            data-testid="user-token-reveal-copy"
          >
            Copy
          </CopyButton>
        </div>
        <p className="text-xs text-muted-foreground">
          Anyone with this token can use the API as you. If you lose the token, revoke it and create a new one.
        </p>
        <DialogFooter>
          <Button onClick={panel.dismissRevealedToken} data-testid="user-token-reveal-done">
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RevokeTokenDialog({ panel }: { panel: PersonalTokensPanel }) {
  const target = panel.revokeTarget;

  return (
    <Dialog
      open={!!target}
      onOpenChange={(open) => {
        if (!open) panel.cancelRevoke();
      }}
    >
      <DialogContent showCloseButton={!panel.isRevoking}>
        <DialogHeader>
          <DialogTitle>Revoke &quot;{target?.name}&quot;?</DialogTitle>
          <DialogDescription>
            Every script or tool that uses this token stops working immediately. You cannot undo this action.
          </DialogDescription>
        </DialogHeader>
        {panel.revokeError && (
          <p className="text-sm text-red-600 dark:text-red-400" role="alert">
            {panel.revokeError}
          </p>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={panel.cancelRevoke} disabled={panel.isRevoking}>
            Keep token
          </Button>
          <LoadingButton
            type="button"
            variant="destructive"
            onClick={() => void panel.confirmRevoke()}
            loading={panel.isRevoking}
            loadingText="Revoking..."
            data-testid="user-token-revoke-confirm"
          >
            <Icon name="trash-2" size="sm" />
            Revoke token
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
