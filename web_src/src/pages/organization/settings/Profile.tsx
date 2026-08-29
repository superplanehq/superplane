import { useState } from "react";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Avatar } from "@/components/Avatar/avatar";
import { Heading } from "@/components/Heading/heading";
import { Icon } from "@/components/Icon";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { useMe } from "@/hooks/useMe";
import { useAccount } from "@/contexts/useAccount";
import { ChangePasswordDialog } from "./components/ChangePasswordDialog";
import { ProfileApiTokensSection } from "./ProfileApiTokensSection";
import { settingsCardClassName } from "./settingsPageStyles";

export function Profile() {
  usePageTitle(["Profile"]);
  const organizationId = useOrganizationId();
  const { data: user, isLoading: loading, error: meError } = useMe();
  const { account } = useAccount();
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  const errorMessage = meError instanceof Error ? meError.message : meError ? "Failed to load profile" : null;

  useReportPageReady(!loading, {
    failed: !!errorMessage,
  });

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
        <div className={settingsCardClassName}>
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
                  onClick={() => setPasswordModalOpen(true)}
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

        <ProfileApiTokensSection organizationId={organizationId} />
      </div>

      {canChangePassword && <ChangePasswordDialog open={passwordModalOpen} onOpenChange={setPasswordModalOpen} />}
    </div>
  );
}
