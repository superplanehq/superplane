import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../../components/Button/button';
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
    } catch (err) {
      setError('Failed to load organizations');
    } finally {
      setLoading(false);
    }
  };

  const handleOrganizationSelect = (org: Organization) => {
    // Redirect to the organization's app route
    window.location.href = `/app/${org.id}`;
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <Text className="text-gray-600 dark:text-gray-400">Loading...</Text>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800 px-4">
      <div className="max-w-2xl w-full bg-white dark:bg-zinc-900 rounded-lg shadow-xl p-8">
        <div className="text-center mb-8">
          <div className="text-4xl mb-4">🛩️</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            Welcome{account?.name ? `, ${account.name}` : ''}!
          </h1>
          <Text className="text-gray-600 dark:text-gray-400">
            Choose an organization to continue, or create a new one.
          </Text>
        </div>

        {error && (
          <div className="mb-6 p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
            <Text className="text-red-700 dark:text-red-400 text-sm">
              {error}
            </Text>
          </div>
        )}

        {organizations.length > 0 ? (
          <div className="space-y-4 mb-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Your Organizations
            </h3>
            {organizations.map((org) => (
              <div
                key={org.id}
                className="flex items-center justify-between p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-blue-300 dark:hover:border-blue-600 transition-colors"
              >
                <div className="flex-1">
                  <h4 className="text-base font-medium text-gray-900 dark:text-white">
                    {org.display_name}
                  </h4>
                  <Text className="text-sm text-gray-600 dark:text-gray-400">
                    {org.description || org.name}
                  </Text>
                </div>
                <Button
                  onClick={() => handleOrganizationSelect(org)}
                  className="ml-4"
                >
                  Join
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 mb-6">
            <Text className="text-gray-600 dark:text-gray-400">
              You're not a member of any organizations yet.
            </Text>
            <Text className="text-gray-600 dark:text-gray-400">
              Create a new organization to get started!
            </Text>
          </div>
        )}

        <div className="border-t border-gray-200 dark:border-gray-700 pt-6">
          <Button
            onClick={() => navigate('/create')}
            outline
            className="w-full"
          >
            Create New Organization
          </Button>
        </div>
      </div>
    </div>
  );
};

export default OrganizationSelect;