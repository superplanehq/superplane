import { useState, type ChangeEvent, type FocusEvent, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { getApiErrorMessage } from "@/lib/errors";
import type { AssignableRoleOption } from "./MemberRowControls";
import { settingsErrorClassName } from "./settingsPageStyles";
import type { InvitationManager } from "./useOrganizationInvitationManager";

interface InviteMemberDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  manager: InvitationManager;
  roles: AssignableRoleOption[];
  rolesLoading: boolean;
  onCreated: () => void;
}

export function InviteMemberDialog({
  open,
  onOpenChange,
  manager,
  roles,
  rolesLoading,
  onCreated,
}: InviteMemberDialogProps) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("org_operator");
  const selectedRole = manager.canAssignRoles ? role : "org_operator";
  const duration = manager.invitations.data?.invitationDurationDays ?? 7;
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (rolesLoading || !email.trim()) return;
    if (!(await manager.createInvitation({ email: email.trim(), role: selectedRole }))) return;
    setEmail("");
    onOpenChange(false);
    onCreated();
  };
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Invite member</DialogTitle>
          <DialogDescription>Send an invitation by email, then copy its link to share it.</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex min-w-0 flex-col gap-5">
          <div className="flex flex-col gap-2">
            <Label htmlFor="invitation-email">Email address</Label>
            <Input
              id="invitation-email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setEmail(event.target.value)}
              disabled={manager.busy}
              className="h-11 text-base sm:h-8 sm:text-sm"
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label htmlFor="invitation-role">Role</Label>
            <Select
              value={selectedRole}
              onValueChange={setRole}
              disabled={!manager.canAssignRoles || rolesLoading || manager.busy}
            >
              <SelectTrigger
                id="invitation-role"
                className="w-full data-[size=default]:h-11 sm:data-[size=default]:h-8"
              >
                <SelectValue placeholder="Operator" />
              </SelectTrigger>
              <SelectContent>
                {roles.map((option) => {
                  const name = option.metadata?.name;
                  if (!name) return null;
                  return (
                    <SelectItem key={name} value={name}>
                      {option.spec?.displayName || name}
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>
          <p className="text-sm text-muted-foreground">
            Invitations expire after {duration} {duration === 1 ? "day" : "days"}.
          </p>
          <InvitationCapacityNotice manager={manager} />
          <InvitationErrors manager={manager} />
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline" className="h-11 sm:h-8">
                Cancel
              </Button>
            </DialogClose>
            <Button
              type="submit"
              className="h-11 sm:h-8"
              disabled={manager.sendingBlocked || manager.busy || rolesLoading || !email.trim()}
            >
              {manager.createPending ? "Sending invitation..." : "Send invitation"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function InvitationErrors({ manager }: { manager: InvitationManager }) {
  const error =
    manager.actionError ||
    (manager.invitations.error &&
      getApiErrorMessage(manager.invitations.error, "Could not load invitations. Refresh the page to try again."));
  if (!error) return null;
  return (
    <p role="alert" className={settingsErrorClassName}>
      {error}
    </p>
  );
}

export function InvitationCapacityNotice({ manager }: { manager: InvitationManager }) {
  const seats = manager.invitations.data?.seats;
  if (!seats || !manager.canInvite || !manager.capacityFull) return null;
  return (
    <p className="text-sm text-amber-700 dark:text-amber-400">
      {seats.perSeat
        ? "All purchased seats are assigned or reserved. Purchase seats in Billing to invite another member."
        : "The member limit is reached. Remove a member or contact a site admin to increase the limit."}
    </p>
  );
}

export function InvitationLinkDialog({
  manager,
  open,
  onOpenChange,
}: {
  manager: InvitationManager;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const link = manager.visibleLink;
  return (
    <Dialog open={open && !!link && manager.canInvite} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Invitation link</DialogTitle>
          <DialogDescription className="break-all">Only {link?.email} can accept this invitation.</DialogDescription>
        </DialogHeader>
        <Label htmlFor="invitation-link" className="sr-only">
          Invitation link for {link?.email}
        </Label>
        <Input
          id="invitation-link"
          readOnly
          value={link?.url ?? ""}
          className="h-11 min-w-0 text-base sm:h-8 sm:text-sm"
          onFocus={(event: FocusEvent<HTMLInputElement>) => event.target.select()}
        />
        {manager.actionError && (
          <p role="alert" className={settingsErrorClassName}>
            {manager.actionError}
          </p>
        )}
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" className="h-11 sm:h-8">
              Done
            </Button>
          </DialogClose>
          <Button onClick={() => manager.copyLink()} disabled={manager.busy} className="h-11 sm:h-8">
            Copy link
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
