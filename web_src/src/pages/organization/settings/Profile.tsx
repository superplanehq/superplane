import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "@/hooks/usePageTitle";
import { meRegenerateToken } from "@/api-client/sdk.gen";
import { Avatar } from "@/components/Avatar/avatar";
import { Heading } from "@/components/Heading/heading";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input as ShadcnInput } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { meKeys, useMe } from "@/hooks/useMe";
import { useAccount } from "@/contexts/AccountContext";
import { showErrorToast, showSuccessToast } from "@/lib/toast.ts";

const MIN_PASSWORD_LENGTH = 8;

export function Profile() {
  usePageTitle(["Profile"]);
  const queryClient = useQueryClient();
  const organizationId = useOrganizationId();
  const { data: user, isLoading: loading, error: meError } = useMe();
  const { account } = useAccount();
  const [actionError, setActionError] = useState<string | null>(null);
  const [token, setToken] = useState<string>("");
  const [tokenVisible, setTokenVisible] = useState(false);
  const [regeneratingToken, setRegeneratingToken] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const [passwordFormError, setPasswordFormError] = useState<string | null>(null);

  const errorMessage =
    actionError || (meError instanceof Error ? meError.message : meError ? "Failed to load profile" : null);

  const resetPasswordForm = () => {
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setPasswordFormError(null);
  };

  const closePasswordModal = () => {
    if (passwordSubmitting) return;
    setPasswordModalOpen(false);
    resetPasswordForm();
  };

  const handleChangePassword = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPasswordFormError(null);

    if (!currentPassword || !newPassword || !confirmPassword) {
      setPasswordFormError("All fields are required.");
      return;
    }

    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setPasswordFormError(`New password must be at least ${MIN_PASSWORD_LENGTH} characters long.`);
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordFormError("New password and confirmation do not match.");
      return;
    }

    if (newPassword === currentPassword) {
      setPasswordFormError("New password must be different from the current password.");
      return;
    }

    setPasswordSubmitting(true);
    try {
      const response = await fetch("/account/password", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ currentPassword, newPassword }),
      });

      if (response.status === 204) {
        resetPasswordForm();
        setPasswordModalOpen(false);
        showSuccessToast("Password updated. Other sessions have been signed out.");
        return;
      }

      const message = (await response.text()).trim();
      switch (response.status) {
        case 401:
          setPasswordFormError(message || "Current password is incorrect.");
          break;
        case 403:
          setPasswordFormError(message || "Password change is unavailable for this account.");
          break;
        case 400:
          setPasswordFormError(message || "Invalid request.");
          break;
        default:
          setPasswordFormError(message || "Failed to update password.");
      }
    } catch (err) {
      setPasswordFormError(err instanceof Error ? err.message : "Failed to update password.");
    } finally {
      setPasswordSubmitting(false);
    }
  };

  const handleRegenerateToken = async () => {
    try {
      setActionError(null);
      setRegeneratingToken(true);
      const response = await meRegenerateToken(withOrganizationHeader({ organizationId }));
      setToken(response.data.token || "");
      setTokenVisible(true);
      queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId!, true) });
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to regenerate token");
    } finally {
      setRegeneratingToken(false);
    }
  };

  const copyToken = async () => {
    if (!token) return;

    try {
      await navigator.clipboard.writeText(token);
      showSuccessToast("API token copied.");
    } catch {
      showErrorToast("Failed to copy API token.");
    }
  };

  if (loading) {
    return (
      <div className="pt-6">
        <div className="flex items-center justify-center py-8">
          <Text className="text-gray-500 dark:text-gray-400">Loading profile...</Text>
        </div>
      </div>
    );
  }

  if (errorMessage) {
    return (
      <div className="pt-6">
        <div className="flex items-center justify-center py-8">
          <Text className="text-red-500">{errorMessage}</Text>
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="pt-6">
        <div className="flex items-center justify-center py-8">
          <Text className="text-gray-500 dark:text-gray-400">No user data available</Text>
        </div>
      </div>
    );
  }

  const canChangePassword = account?.has_password === true;

  return (
    <div className="pt-6 max-w-none">
      <Heading level={2} className="text-lg font-medium text-left text-gray-800 dark:text-white mb-4">
        Profile Information
      </Heading>
      <div className="space-y-6">
        {/* Profile Section */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-700 p-6">
          <div className="space-y-6">
            {/* User Avatar and Basic Info */}
            <div className="flex items-center space-x-4">
              <Avatar
                initials={user.email ? user.email.charAt(0).toUpperCase() : "U"}
                alt="User Avatar"
                className="w-16 h-16"
              />
              <div>
                <Heading level={3} className="text-lg font-medium text-gray-800 dark:text-white">
                  {user.email}
                </Heading>
              </div>
            </div>

            {/* User Information */}
            <div className="space-y-4">
              <div>
                <Text className="text-sm text-left font-medium text-gray-800 dark:text-gray-300">User ID</Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">{user.id}</Text>
              </div>
              <div>
                <Text className="text-sm text-left font-medium text-gray-800 dark:text-gray-300">Email Address</Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">{user.email}</Text>
              </div>

              <div>
                <Text className="text-sm text-left font-medium text-gray-800 dark:text-gray-300">Member Since</Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">
                  {user.createdAt ? new Date(user.createdAt).toLocaleDateString() : "Not available"}
                </Text>
              </div>
            </div>

            {canChangePassword && (
              <div className="flex items-center gap-4">
                <Button
                  type="button"
                  onClick={() => {
                    resetPasswordForm();
                    setPasswordModalOpen(true);
                  }}
                  className="flex items-center gap-2"
                  data-testid="change-password-button"
                >
                  <Icon name="lock" />
                  Change password
                </Button>
              </div>
            )}
          </div>
        </div>

        <Heading level={2} className="text-lg text-left font-medium text-gray-800 dark:text-white mb-0">
          API Token
        </Heading>
        <Text className="text-gray-800 text-left dark:text-gray-400 text-sm">
          Use this token to authenticate API requests to SuperPlane. Keep your token secure and do not share it.
        </Text>

        {/* API Token Section */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-700 p-6">
          <div className="space-y-4">
            {/* Token Status */}
            {!user.hasToken && (
              <div className="flex items-center gap-2">
                <Icon name="key-round" className="text-gray-500 dark:text-gray-400 text-lg" />
                <Text className="text-sm font-medium text-gray-500 dark:text-gray-400">No API token generated</Text>
              </div>
            )}

            <div className="flex items-center gap-4">
              <LoadingButton
                onClick={handleRegenerateToken}
                loading={regeneratingToken}
                loadingText="Regenerating..."
                className="flex items-center gap-2"
              >
                <Icon name="refresh-ccw" />
                {user.hasToken ? "Regenerate Token" : "Generate Token"}
              </LoadingButton>

              {user.hasToken && !token && (
                <Text className="text-gray-500 dark:text-gray-400 text-sm">
                  Your current token is hidden for security. Generate a new token to view it.
                </Text>
              )}
            </div>

            {token && (
              <div className="space-y-3">
                <Text className="text-sm font-medium text-gray-700 dark:text-gray-300">New API Token</Text>
                <div className="flex items-center gap-2 ph-no-capture">
                  <Input
                    type={tokenVisible ? "text" : "password"}
                    value={token}
                    readOnly
                    className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-900"
                  />
                  <Button
                    variant="outline"
                    onClick={() => setTokenVisible(!tokenVisible)}
                    className="flex items-center gap-1"
                  >
                    <Icon name={tokenVisible ? "eye-closed" : "eye"} />
                  </Button>
                  <Button variant="outline" onClick={copyToken} className="flex items-center gap-1">
                    <Icon name="copy" />
                    Copy
                  </Button>
                </div>
                <div className="bg-orange-50 dark:bg-amber-900/20 border border-amber-950/15 dark:border-amber-100/15 rounded-lg p-3">
                  <div className="flex items-start gap-2">
                    <Icon name="key-round" className="text-amber-800 dark:text-amber-400 text-sm mt-0.5" />
                    <Text className="text-amber-800 dark:text-amber-200 text-sm">
                      <strong>Important:</strong> This token will only be shown once. Make sure to copy and store it
                      securely. If you lose this token, you'll need to generate a new one.
                    </Text>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <Dialog
        open={passwordModalOpen}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            closePasswordModal();
            return;
          }
          setPasswordModalOpen(true);
        }}
      >
        <DialogContent showCloseButton={!passwordSubmitting}>
          <DialogHeader>
            <DialogTitle>Change password</DialogTitle>
            <DialogDescription>
              Update the password for your SuperPlane account. This will sign you out of every other session and revoke
              all API tokens issued for your account.
            </DialogDescription>
          </DialogHeader>
          <form className="space-y-4" onSubmit={handleChangePassword}>
            <div className="space-y-2">
              <Label htmlFor="profile-current-password">Current password</Label>
              <ShadcnInput
                id="profile-current-password"
                type="password"
                autoComplete="current-password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                disabled={passwordSubmitting}
                className="ph-no-capture"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="profile-new-password">New password</Label>
              <ShadcnInput
                id="profile-new-password"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                disabled={passwordSubmitting}
                minLength={MIN_PASSWORD_LENGTH}
                className="ph-no-capture"
                required
              />
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Must be at least {MIN_PASSWORD_LENGTH} characters long.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="profile-confirm-password">Confirm new password</Label>
              <ShadcnInput
                id="profile-confirm-password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                disabled={passwordSubmitting}
                minLength={MIN_PASSWORD_LENGTH}
                className="ph-no-capture"
                required
              />
            </div>
            {passwordFormError && (
              <p className="text-sm text-red-600 dark:text-red-400" role="alert">
                {passwordFormError}
              </p>
            )}
            <DialogFooter className="mt-4">
              <Button type="button" variant="outline" onClick={closePasswordModal} disabled={passwordSubmitting}>
                Cancel
              </Button>
              <LoadingButton
                type="submit"
                loading={passwordSubmitting}
                loadingText="Updating..."
                data-testid="change-password-submit"
              >
                Update password
              </LoadingButton>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
