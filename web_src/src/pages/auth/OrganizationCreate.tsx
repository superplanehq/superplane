import React, { useState } from "react";
import { Link } from "react-router-dom";
import { Text } from "../../components/Text/text";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { LoadingButton } from "@/components/ui/loading-button";
import { getUsageLimitNotice } from "@/utils/usageLimits";
import { getResponseErrorMessage } from "@/utils/errors";

const OrganizationCreate: React.FC = () => {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
        const fallbackMessage =
          response.status === 409 ? "An organization with this name already exists" : "Failed to create organization";
        const errorMessage = await getResponseErrorMessage(response, fallbackMessage);
        setError(errorMessage);
      }
    } catch (err) {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const usageLimitNotice = error ? getUsageLimitNotice(error) : null;

  return (
    <div className="min-h-screen bg-slate-100">
      <div className="p-6 flex items-center">
        <Link to="/" className="text-sm font-medium text-gray-500 px-2 py-1 hover:bg-gray-950/5 rounded">
          ← Back to Organizations
        </Link>
      </div>
      <div className="flex items-center justify-center p-8">
        <div className="max-w-md w-full bg-white rounded-lg shadow-sm p-8 outline outline-slate-950/10">
          <div className="text-center mb-8">
            <h4 className="text-xl font-semibold text-gray-800 mb-1">Create Organization</h4>
            <Text className="text-gray-800">Set up a new SuperPlane organization</Text>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            {usageLimitNotice ? <UsageLimitAlert notice={usageLimitNotice} /> : null}
            {error && !usageLimitNotice ? (
              <Alert variant="destructive">
                <AlertTitle>Unable to create organization</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}

            <div>
              <label
                htmlFor="name"
                className="block text-sm font-medium text-gray-800 text-left dark:text-gray-300 mb-2"
              >
                Organization Name
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full px-3 py-2 outline-1 outline-slate-300 rounded-md shadow-md focus:outline-gray-800"
                placeholder="e.g. Super Duper Org"
                data-1p-ignore
              />
            </div>

            <div className="flex space-x-4">
              <LoadingButton
                type="submit"
                className="flex-1"
                disabled={!name.trim()}
                loading={loading}
                loadingText="Creating..."
              >
                Create Organization
              </LoadingButton>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default OrganizationCreate;
