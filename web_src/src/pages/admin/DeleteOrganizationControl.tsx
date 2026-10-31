import { useState } from "react";
import { useNavigate } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
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
import { accountOrganizationsQueryKey } from "@/hooks/useAccountOrganizations";
import { showSuccessToast } from "@/lib/toast";
import { getResponseErrorMessage } from "@/lib/errors";

export function DeleteOrganizationControl({ organization }: { organization: { id: string; name: string } }) {
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const changeOpen = (nextOpen: boolean) => {
    if (deleting) return;
    setOpen(nextOpen);
    setConfirmation("");
    setError("");
  };

  const remove = async () => {
    if (deleting || confirmation !== organization.name) return;
    setDeleting(true);
    setError("");
    try {
      const response = await fetch(`/admin/api/organizations/${organization.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!response.ok) {
        setError(await getResponseErrorMessage(response, "Could not delete the organization. Try again."));
        setDeleting(false);
        return;
      }
    } catch {
      setError("Could not delete the organization. Check your connection and try again.");
      setDeleting(false);
      return;
    }
    void queryClient.invalidateQueries({ queryKey: accountOrganizationsQueryKey });
    showSuccessToast("Organization deleted.");
    navigate("/admin", { replace: true });
  };

  return (
    <>
      <Button variant="destructive" size="sm" onClick={() => changeOpen(true)}>
        Delete organization
      </Button>
      <Dialog open={open} onOpenChange={changeOpen}>
        <DialogContent showCloseButton={!deleting}>
          <DialogHeader>
            <DialogTitle>Delete “{organization.name}”?</DialogTitle>
            <DialogDescription>
              This deletes the organization, its workspaces, and its automations. You cannot undo this action. Member
              accounts remain available.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="delete-organization-name">Organization name</Label>
            <p id="delete-organization-help" className="text-sm text-muted-foreground">
              Type {organization.name} to confirm deletion.
            </p>
            <Input
              id="delete-organization-name"
              aria-describedby="delete-organization-help"
              value={confirmation}
              onChange={(event) => setConfirmation(event.target.value)}
              disabled={deleting}
              autoComplete="off"
            />
          </div>
          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button variant="outline" disabled={deleting} onClick={() => changeOpen(false)}>
              Keep organization
            </Button>
            <Button
              variant="destructive"
              disabled={deleting || confirmation !== organization.name}
              onClick={() => void remove()}
            >
              {deleting ? "Deleting..." : "Delete organization"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
