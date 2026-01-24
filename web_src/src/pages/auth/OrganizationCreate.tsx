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
    <div className="min-h-screen bg-slate-100 dark:bg-neutral-900">
      <div className="p-6 flex items-center">
        <button
          type="button"
          onClick={() => navigate("/")}
          className="text-sm font-medium text-gray-500 dark:text-gray-400 px-2 py-1 hover:bg-gray-950/5 dark:hover:bg-white/5 rounded"
        >
          ← Back to Organizations
        </button>
      </div>
      <div className="flex items-center justify-center p-8">
        <div className="max-w-md w-full bg-white dark:bg-neutral-800 rounded-lg shadow-sm p-8 outline outline-slate-950/10 dark:outline-neutral-700">
          <div className="text-center mb-8">
            <h4 className="text-xl font-semibold text-gray-800 dark:text-white mb-1">Create Organization</h4>
            <Text className="text-gray-800 dark:text-gray-300">Set up a new SuperPlane organization</Text>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
              <div className="p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
                <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
              </div>
            )}

            <div>
              <label
                htmlFor="name"
                className="block text-sm font-medium text-gray-800 dark:text-gray-200 text-left mb-2"
              >
                Organization Name
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full px-3 py-2 outline-1 outline-slate-300 dark:outline-neutral-600 rounded-md shadow-md focus:outline-gray-800 dark:focus:outline-neutral-400 bg-white dark:bg-neutral-700 text-gray-900 dark:text-white"
                placeholder="e.g. Super Duper Org"
                data-1p-ignore
              />
            </div>

            <div className="flex space-x-4">
              <Button type="submit" className="flex-1" disabled={loading || !name.trim()}>
                {loading ? "Creating..." : "Create Organization"}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default OrganizationCreate;
