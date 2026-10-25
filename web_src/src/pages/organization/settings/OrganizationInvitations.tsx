import { Link } from "react-router";
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
import { organizationSettingsSectionPath } from "@/pages/factories/lib/factoryPagePaths";
import type { AssignableRoleOption } from "./MemberRowControls";
import { settingsErrorClassName } from "./settingsPageStyles";
import type { InvitationCapacity, InvitationManager } from "./useOrganizationInvitationManager";

interface InviteMemberDialogProps {
  organizationId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  manager: InvitationManager;
  roles: AssignableRoleOption[];
  rolesLoading: boolean;
  onCreated: () => void;
}

export function InviteMemberDialog({
  organizationId,
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
          <InvitationCapacityNotice manager={manager} organizationId={organizationId} />
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

type InvitationCapacityNoticeManager = Pick<
  InvitationManager,
  "capacity" | "canInvite" | "canManageBilling" | "canPurchaseSeats"
>;

export function InvitationCapacityNotice({
  manager,
  organizationId,
}: {
  manager: InvitationCapacityNoticeManager;
  organizationId: string;
}) {
  const capacity = manager.capacity;
  if (!capacity || !manager.canInvite) return null;
  const message = capacityMessage(capacity, manager);
  if (!message) return null;
  const billingPath = organizationSettingsSectionPath(organizationId, "billing");
  const action = capacityAction(capacity, manager, billingPath);
  return (
    <div role="status" aria-label="Invitation capacity" className="text-sm text-amber-700 dark:text-amber-400">
      {message}
      {action && (
        <>
          {" "}
          <Link to={action.to} className="font-medium underline underline-offset-2">
            {action.label}
          </Link>
        </>
      )}
    </div>
  );
}

function capacityMessage(capacity: InvitationCapacity, manager: InvitationCapacityNoticeManager): string | null {
  const { canManageBilling, canPurchaseSeats } = manager;
  switch (capacity.state) {
    case "paymentBlocked":
      return canManageBilling
        ? "Invitations are paused until the payment method is updated."
        : "Invitations are paused until the payment method is updated. Ask a billing manager to update it.";
    case "overCapacity":
      return `${capacity.members} members use ${capacity.paidSeats} paid seats. ${
        canPurchaseSeats ? "Purchase seats to invite another member." : "Remove a member to invite another member."
      }`;
    case "purchaseRequired":
      return canManageBilling
        ? "All purchased seats are assigned or reserved. Purchase seats to invite another member."
        : "All purchased seats are assigned or reserved. Ask a billing manager to purchase seats.";
    case "planMaximum":
      return "Remove a member or cancel a pending invitation to free a seat.";
    case "memberLimit":
      return "The member limit is reached. Remove a member or contact a site admin to increase the limit.";
    case "available":
    case "unlimited":
      return null;
  }
}

function capacityAction(
  capacity: InvitationCapacity,
  manager: InvitationCapacityNoticeManager,
  billingPath: string,
): { label: string; to: string } | null {
  if (capacity.state === "paymentBlocked" && manager.canManageBilling) {
    return { label: "Open Billing", to: billingPath };
  }
  const purchasable = capacity.state === "purchaseRequired" || capacity.state === "overCapacity";
  if (purchasable && manager.canPurchaseSeats) {
    return { label: "Purchase seats", to: `${billingPath}?buySeats=1` };
  }
  return null;
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
