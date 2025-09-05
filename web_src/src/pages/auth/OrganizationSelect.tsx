import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Text } from '../../components/Text/text';
import { useAccount } from '../../contexts/AccountContext';

interface Organization {
  id: string;
  name: string;
  display_name: string;
  description?: string;
}

const OrganizationSelect: React.FC = () => {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { account } = useAccount();
  const navigate = useNavigate();

  useEffect(() => {
    fetchOrganizations();
  }, [account]);

  const fetchOrganizations = async () => {
    if (!account) return;

    try {
      const orgsResponse = await fetch('/organizations', {
        credentials: 'include',
      });

      if (orgsResponse.ok) {
        const organizations = await orgsResponse.json();
        setOrganizations(organizations);
      } else {
        setError('Failed to load organizations');
      }
    } catch {
      setError('Failed to load organizations');
    } finally {
      setLoading(false);
    }
  };

  const handleOrganizationSelect = (org: Organization) => {
    // Redirect to the organization's app route
    window.location.href = `/${org.id}`;
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800 p-8">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <Text className="text-gray-600 dark:text-gray-400">Loading...</Text>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800">
      <div className="p-8 pt-6">
        <div className="max-w-7xl w-full">
          <div className="mb-4">
            <Text className="text-2xl font-bold text-gray-700 dark:text-gray-300 text-left">
              Hey there{account?.name ? `, ${account.name}` : ''}! How's it going?
            </Text>
            {organizations.length > 0 && (
              <Text className="text-2xl font-bold text-gray-700 dark:text-gray-300 text-left">
                Select one of your organizations below to get started:
              </Text>
            )}
          </div>

          {error && (
            <div className="mb-6 p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 dark:text-red-400 text-sm">
                {error}
              </Text>
            </div>
          )}

          {organizations.length === 0 && (
            <div className="text-left py-2 mb-4">
              <Text className="text-gray-600 dark:text-gray-400">
                You're not a member of any organizations yet.
              </Text>
              <Text className="text-gray-600 dark:text-gray-400">
                Create a new organization to get started!
              </Text>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {organizations.map((org) => (
              <div
                key={org.id}
                className="bg-white dark:bg-zinc-900 rounded-lg shadow-md p-6 border border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600 transition-colors cursor-pointer"
                onClick={() => handleOrganizationSelect(org)}
              >
                <div className="flex items-center mb-2">
                  <span className="text-md mr-2">🏢</span>
                  <h4 className="text-md font-semibold text-gray-900 dark:text-white truncate">
                    {org.display_name}
                  </h4>
                </div>
                <Text className="text-sm text-left text-gray-600 dark:text-gray-400 truncate">
                  {org.description}
                </Text>
              </div>
            ))}

            <div
              className="bg-blue-50 dark:bg-blue-900/20 border-2 border-dashed border-blue-300 dark:border-blue-600 rounded-lg p-6 flex flex-col items-center justify-center hover:bg-blue-100 dark:hover:bg-blue-900/30 transition-colors cursor-pointer"
              onClick={() => navigate('/create')}
            >
              <div className="flex items-center mb-1">
                <div className="text-2xl mb-3 mr-2 text-blue-600 dark:text-blue-400">+</div>
                <h4 className="text-lg font-semibold text-blue-900 dark:text-blue-200 mb-2 text-center">
                  Create New
                </h4>
              </div>
              <Text className="text-sm text-blue-500 dark:text-blue-300 text-center">
                Start fresh with a new organization
              </Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default OrganizationSelect;