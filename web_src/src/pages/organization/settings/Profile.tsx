import { useAccount } from '../../../contexts/AccountContext';
import { Avatar } from '../../../components/Avatar/avatar';
import { Text } from '../../../components/Text/text';
import { Heading } from '../../../components/Heading/heading';

export function Profile() {
  const { account: user, loading } = useAccount();
  const error = null; // AccountContext doesn't expose error state

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
                {/* Created date not available in AccountContext */}
                Not available
              </Text>
            </div>
          </div>

          {/* Note: Account providers not available in simplified account context */}
        </div>
      </div>
    </div>
  );
}