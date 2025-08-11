import { useUserStore } from '../../../stores/userStore';
import { Avatar } from '../../../components/Avatar/avatar';
import { Text } from '../../../components/Text/text';
import { Heading } from '../../../components/Heading/heading';
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol';

export function Profile() {
  const { user, loading, error } = useUserStore();

  if (loading) {
    return (
      <div className="pt-6">
        <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">
          Profile Settings
        </Heading>
        <div className="flex items-center justify-center py-8">
          <Text className="text-zinc-500 dark:text-zinc-400">Loading profile...</Text>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="pt-6">
        <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">
          Profile Settings
        </Heading>
        <div className="flex items-center justify-center py-8">
          <Text className="text-red-500">{error}</Text>
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="pt-6">
        <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">
          Profile Settings
        </Heading>
        <div className="flex items-center justify-center py-8">
          <Text className="text-zinc-500 dark:text-zinc-400">No user data available</Text>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-6">
      <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">
        Profile Settings
      </Heading>

      <div className="bg-white dark:bg-zinc-800 rounded-lg shadow-sm border border-zinc-200 dark:border-zinc-700 p-6">
        <div className="space-y-6">
          {/* User Avatar and Basic Info */}
          <div className="flex items-center space-x-4">
            <Avatar
              src={user.avatar_url}
              initials={user.name ? user.name.split(' ').map(n => n[0]).join('').toUpperCase() : 'U'}
              alt={user.name}
              className="w-16 h-16"
            />
            <div>
              <Heading level={2} className="text-lg font-medium text-zinc-900 dark:text-white">
                {user.name}
              </Heading>
              <Text className="text-zinc-500 dark:text-zinc-400">
                {user.email}
              </Text>
            </div>
          </div>

          {/* User Information */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <Text className="text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                User ID
              </Text>
              <Text className="text-zinc-900 dark:text-white font-mono text-sm bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded">
                {user.id}
              </Text>
            </div>

            <div>
              <Text className="text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Member Since
              </Text>
              <Text className="text-zinc-900 dark:text-white">
                {new Date(user.created_at).toLocaleDateString()}
              </Text>
            </div>
          </div>

          {/* Account Providers */}
          <div>
            <Text className="text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-3">
              Connected Accounts
            </Text>
            {user.account_providers && user.account_providers.length > 0 ? (
              <div className="space-y-2">
                {user.account_providers.map((provider) => (
                  <div key={provider.id} className="flex items-center space-x-3 p-3 bg-zinc-50 dark:bg-zinc-700 rounded-md">
                    <MaterialSymbol name="account_circle" className="text-zinc-500 dark:text-zinc-400" />
                    <div className="flex-1">
                      <Text className="text-sm font-medium text-zinc-900 dark:text-white">
                        {provider.name || provider.username || provider.email || 'Unknown'}
                      </Text>
                      <Text className="text-xs text-zinc-500 dark:text-zinc-400">
                        {provider.provider} • Connected {new Date(provider.created_at).toLocaleDateString()}
                      </Text>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="p-3 bg-zinc-50 dark:bg-zinc-700 rounded-md">
                <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                  No connected accounts found
                </Text>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}