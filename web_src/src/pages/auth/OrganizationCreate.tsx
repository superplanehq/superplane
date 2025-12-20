import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Text } from "../../components/Text/text";
import { Button } from "../../ui/button";

const OrganizationCreate: React.FC = () => {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/organizations", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          name: name.trim(),
        }),
      });

      if (response.ok) {
        const org = await response.json();
        // Redirect to the new organization
        window.location.href = `/${org.id}`;
      } else {
        try {
          const errorData = await response.json();
          setError(errorData.message || "Failed to create organization");
        } catch {
          // If we can't parse the error response, show a generic message based on status
          if (response.status === 409) {
            setError("An organization with this name already exists");
          } else {
            setError(`Failed to create organization (${response.status})`);
          }
        }
      }
    } catch (err) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800 px-4">
      <div className="max-w-lg w-full bg-white dark:bg-gray-900 rounded-lg shadow-xl p-8">
        <div className="text-center mb-8">
          <div className="text-4xl mb-4">🏢</div>
          <h4 className="text-2xl font-bold text-gray-800 dark:text-white mb-2">Create Organization</h4>
          <Text className="text-gray-600 dark:text-gray-400">Set up a new organization</Text>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
            </div>
          )}

          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 text-left dark:text-gray-300 mb-2">
              Name
            </label>
            <input
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-800 dark:text-white"
              placeholder="Acme Corporation"
            />
          </div>

          <div className="flex space-x-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => navigate("/")}
              className="flex-1"
              disabled={loading}
            >
              Cancel
            </Button>

            <Button type="submit" className="flex-1" disabled={loading || !name.trim()}>
              {loading ? "Creating..." : "Create Organization"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default OrganizationCreate;
