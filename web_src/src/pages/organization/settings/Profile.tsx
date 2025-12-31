import { useEffect, useState } from "react";
import { meMe, meRegenerateToken } from "../../../api-client/sdk.gen";
import type { SuperplaneMeUser } from "../../../api-client/types.gen";
import { Avatar } from "../../../components/Avatar/avatar";
import { Heading } from "../../../components/Heading/heading";
import { Icon } from "../../../components/Icon";
import { Input } from "../../../components/Input/input";
import { Text } from "../../../components/Text/text";
import { Button } from "../../../ui/button";
import { withOrganizationHeader } from "../../../utils/withOrganizationHeader";

export function Profile() {
  const [user, setUser] = useState<SuperplaneMeUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [token, setToken] = useState<string>("");
  const [tokenVisible, setTokenVisible] = useState(false);
  const [regeneratingToken, setRegeneratingToken] = useState(false);

  useEffect(() => {
    const fetchUserData = async () => {
      try {
        setLoading(true);
        const response = await meMe(withOrganizationHeader());
        setUser(response.data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load profile");
      } finally {
        setLoading(false);
      }
    };

    fetchUserData();
  }, []);

  const handleRegenerateToken = async () => {
    try {
      setRegeneratingToken(true);
      const response = await meRegenerateToken(withOrganizationHeader());
      setToken(response.data.token || "");
      setTokenVisible(true);
      // Update user to reflect token existence
      if (user) {
        setUser({ ...user, hasToken: true });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to regenerate token");
    } finally {
      setRegeneratingToken(false);
    }
  };

  const copyToken = () => {
    navigator.clipboard.writeText(token);
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

  if (error) {
    return (
      <div className="pt-6">
        <div className="flex items-center justify-center py-8">
          <Text className="text-red-500">{error}</Text>
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

  return (
    <div className="pt-6 max-w-none">
      <Heading level={2} className="text-lg font-medium text-left text-gray-800 dark:text-white mb-4">
        Profile Information
      </Heading>
      <div className="space-y-6">
        {/* Profile Section */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
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
                <Text className="text-sm text-left font-medium text-gray-700 dark:text-gray-300 mb-2">User ID</Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">{user.id}</Text>
              </div>
              <div>
                <Text className="text-sm text-left font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Email Address
                </Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">{user.email}</Text>
              </div>

              <div>
                <Text className="text-sm text-left font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Member Since
                </Text>
                <Text className="text-left text-gray-500 dark:text-gray-400">
                  {user.createdAt ? new Date(user.createdAt).toLocaleDateString() : "Not available"}
                </Text>
              </div>
            </div>
          </div>
        </div>

        <Heading level={2} className="text-lg text-left font-medium text-gray-800 dark:text-white mb-4">
          API Token
        </Heading>
        <Text className="text-gray-600 text-left dark:text-gray-400 text-sm">
          Use this token to authenticate API requests to Superplane. Keep your token secure and do not share it.
        </Text>

        {/* API Token Section */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <div className="space-y-4">
            {/* Token Status */}
            <div className="flex items-center gap-2">
              {!user.hasToken && (
                <>
                  <Icon name="error" className="text-gray-500 dark:text-gray-400 text-lg" />
                  <Text className="text-sm font-medium text-gray-600 dark:text-gray-400">No API token generated</Text>
                </>
              )}
            </div>

            <div className="flex items-center gap-4">
              <Button onClick={handleRegenerateToken} disabled={regeneratingToken} className="flex items-center gap-2">
                <Icon name="refresh-ccw" />
                {regeneratingToken ? "Regenerating..." : user.hasToken ? "Regenerate Token" : "Generate Token"}
              </Button>

              {user.hasToken && !token && (
                <Text className="text-gray-500 dark:text-gray-400 text-sm">
                  Your current token is hidden for security. Generate a new token to view it.
                </Text>
              )}
            </div>

            {token && (
              <div className="space-y-3">
                <Text className="text-sm font-medium text-gray-700 dark:text-gray-300">New API Token</Text>
                <div className="flex items-center gap-2">
                  <Input
                    type={tokenVisible ? "text" : "password"}
                    value={token}
                    readOnly
                    className="flex-1 font-mono text-sm bg-gray-50 dark:bg-gray-900"
                  />
                  <Button
                    variant="secondary"
                    onClick={() => setTokenVisible(!tokenVisible)}
                    className="flex items-center gap-1"
                  >
                    <Icon name={tokenVisible ? "visibility_off" : "visibility"} />
                  </Button>
                  <Button variant="secondary" onClick={copyToken} className="flex items-center gap-1">
                    <Icon name="copy" />
                    Copy
                  </Button>
                </div>
                <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
                  <div className="flex items-start gap-2">
                    <Icon name="warning" className="text-amber-600 dark:text-amber-400 text-sm mt-0.5" />
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
    </div>
  );
}
