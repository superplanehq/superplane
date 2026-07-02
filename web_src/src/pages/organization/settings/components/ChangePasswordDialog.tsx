import { useState } from "react";
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
import { showSuccessToast } from "@/lib/toast.ts";

const MIN_PASSWORD_LENGTH = 8;

export interface ChangePasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface FormState {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

const EMPTY_FORM: FormState = {
  currentPassword: "",
  newPassword: "",
  confirmPassword: "",
};

function validate(form: FormState): string | null {
  if (!form.currentPassword || !form.newPassword || !form.confirmPassword) {
    return "All fields are required.";
  }

  if (form.newPassword.length < MIN_PASSWORD_LENGTH) {
    return `New password must be at least ${MIN_PASSWORD_LENGTH} characters long.`;
  }

  if (form.newPassword !== form.confirmPassword) {
    return "New password and confirmation do not match.";
  }

  if (form.newPassword === form.currentPassword) {
    return "New password must be different from the current password.";
  }

  return null;
}

function errorFromResponseStatus(status: number, message: string): string {
  switch (status) {
    case 401:
      return message || "Current password is incorrect.";
    case 403:
      return message || "Password change is unavailable for this account.";
    case 400:
      return message || "Invalid request.";
    default:
      return message || "Failed to update password.";
  }
}

interface PasswordFieldProps {
  id: string;
  label: string;
  autoComplete: string;
  value: string;
  onChange: (next: string) => void;
  disabled: boolean;
  /**
   * Optional minimum length enforced by the browser. We only set this on the
   * "new password" fields. The current-password field must accept whatever
   * the user signed up with, even if it predates the current minimum.
   */
  minLength?: number;
  hint?: string;
}

function PasswordField({ id, label, autoComplete, value, onChange, disabled, minLength, hint }: PasswordFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="password"
        autoComplete={autoComplete}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        minLength={minLength}
        className="ph-no-capture"
        required
      />
      {hint && <p className="text-xs text-gray-500 dark:text-gray-400">{hint}</p>}
    </div>
  );
}

async function submitChangePassword(form: FormState): Promise<string | null> {
  const response = await fetch("/account/password", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      currentPassword: form.currentPassword,
      newPassword: form.newPassword,
    }),
  });

  if (response.status === 204) {
    return null;
  }

  const message = (await response.text()).trim();
  return errorFromResponseStatus(response.status, message);
}

export function ChangePasswordDialog({ open, onOpenChange }: ChangePasswordDialogProps) {
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const reset = () => {
    setForm(EMPTY_FORM);
    setFormError(null);
  };

  const handleClose = () => {
    if (submitting) return;
    onOpenChange(false);
    reset();
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    const validationError = validate(form);
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setSubmitting(true);
    try {
      const error = await submitChangePassword(form);
      if (error) {
        setFormError(error);
        return;
      }

      reset();
      onOpenChange(false);
      showSuccessToast("Password updated. Other sessions have been signed out.");
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to update password.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleClose();
          return;
        }
        onOpenChange(true);
      }}
    >
      <DialogContent showCloseButton={!submitting}>
        <DialogHeader>
          <DialogTitle>Change password</DialogTitle>
          <DialogDescription>
            Update the password for your SuperPlane account. This will sign you out of every other session and revoke
            all API tokens issued for your account.
          </DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={handleSubmit}>
          <PasswordField
            id="profile-current-password"
            label="Current password"
            autoComplete="current-password"
            value={form.currentPassword}
            onChange={(value) => setForm((prev) => ({ ...prev, currentPassword: value }))}
            disabled={submitting}
          />
          <PasswordField
            id="profile-new-password"
            label="New password"
            autoComplete="new-password"
            value={form.newPassword}
            onChange={(value) => setForm((prev) => ({ ...prev, newPassword: value }))}
            disabled={submitting}
            minLength={MIN_PASSWORD_LENGTH}
            hint={`Must be at least ${MIN_PASSWORD_LENGTH} characters long.`}
          />
          <PasswordField
            id="profile-confirm-password"
            label="Confirm new password"
            autoComplete="new-password"
            value={form.confirmPassword}
            onChange={(value) => setForm((prev) => ({ ...prev, confirmPassword: value }))}
            disabled={submitting}
            minLength={MIN_PASSWORD_LENGTH}
          />
          {formError && (
            <p className="text-sm text-red-600 dark:text-red-400" role="alert">
              {formError}
            </p>
          )}
          <DialogFooter className="mt-4">
            <Button type="button" variant="outline" onClick={handleClose} disabled={submitting}>
              Cancel
            </Button>
            <LoadingButton
              type="submit"
              loading={submitting}
              loadingText="Updating..."
              data-testid="change-password-submit"
            >
              Update password
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
