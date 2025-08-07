import React, { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button } from '../../components/Button/button';
import { Field, FieldGroup, Label } from '../../components/Fieldset/fieldset';
import { Input } from '../../components/Input/input';
import { Text } from '../../components/Text/text';

const OrganizationLogin: React.FC = () => {
  const [organizationName, setOrganizationName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!organizationName.trim()) {
      setError('Organization name is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // Check if organization exists by trying to load its login page
      const response = await fetch(`/login/${organizationName}`);
      
      if (response.ok) {
        // Redirect to organization-specific login page
        window.location.href = `/login/${organizationName}`;
      } else if (response.status === 404) {
        setError('Organization not found. Please check the name and try again.');
      } else {
        setError('An error occurred. Please try again.');
      }
    } catch (err) {
      setError('Unable to connect. Please check your internet connection.');
    } finally {
      setLoading(false);
    }
  };

  // Check if we came from an invitation link
  const invitationToken = searchParams.get('invitation');
  const invitedOrg = searchParams.get('org');

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-zinc-900 dark:to-zinc-800 px-4">
      <div className="max-w-md w-full bg-white dark:bg-zinc-900 rounded-lg shadow-xl p-8">
        <div className="text-center mb-8">
          <div className="text-4xl mb-4">🛩️</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            Welcome to SuperPlane
          </h1>
          {invitationToken ? (
            <Text className="text-gray-600 dark:text-gray-400">
              You've been invited to join <strong>{invitedOrg}</strong>
            </Text>
          ) : (
            <Text className="text-gray-600 dark:text-gray-400">
              Sign in to your organization
            </Text>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <FieldGroup>
            <Field>
              <Label htmlFor="organization">Organization Name</Label>
              <Input
                id="organization"
                type="text"
                value={invitedOrg || organizationName}
                onChange={(e) => setOrganizationName(e.target.value)}
                placeholder="your-organization-name"
                disabled={loading || !!invitedOrg}
                autoFocus
                required
              />
            </Field>
          </FieldGroup>

          {error && (
            <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 dark:text-red-400 text-sm">
                {error}
              </Text>
            </div>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={loading}
          >
            {loading ? 'Checking...' : 'Continue'}
          </Button>
        </form>

        <div className="mt-8 text-center">
          <Text className="text-xs text-gray-500 dark:text-gray-400">
            Don't have an organization? Contact your administrator for an invitation.
          </Text>
        </div>
      </div>
    </div>
  );
};

export default OrganizationLogin;