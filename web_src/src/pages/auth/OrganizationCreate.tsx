import React, { useState } from "react";
import { Link } from "react-router-dom";
import { Text } from "../../components/Text/text";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { getUsageLimitNotice } from "@/lib/usageLimits";
import { getResponseErrorMessage } from "@/lib/errors";
import { analytics } from "@/lib/analytics";
import { useReportPageReady } from "@/hooks/useReportPageReady";

const OrganizationCreate: React.FC = () => {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useReportPageReady(true);

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
        analytics.orgCreate(org.id);
        // Redirect to the new organization
        window.location.href = `/${org.id}`;
      } else {
        const fallbackMessage =
          response.status === 409 ? "An organization with this name already exists" : "Failed to create organization";
        const errorMessage = await getResponseErrorMessage(response, fallbackMessage);
        setError(errorMessage);
      }
    } catch {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const usageLimitNotice = error ? getUsageLimitNotice(error) : null;

  return (
    <div className="min-h-screen bg-surface-canvas">
      <div className="flex items-center p-6">
        <Link
          to="/?select=true"
          className="rounded px-2 py-1 text-sm font-medium text-content-secondary hover:bg-surface-subtle"
        >
          ← Back to Organizations
        </Link>
      </div>
      <div className="flex items-center justify-center p-8">
        <div className="w-full max-w-md rounded-lg border border-edge-default bg-surface-raised p-8 shadow-sm">
          <div className="mb-8 text-center">
            <h4 className="mb-1 text-xl font-semibold text-content-primary">Create Organization</h4>
            <Text className="text-content-secondary">Set up a new SuperPlane organization</Text>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            {usageLimitNotice ? <UsageLimitAlert notice={usageLimitNotice} /> : null}
            {error && !usageLimitNotice ? (
              <Alert variant="destructive">
                <AlertTitle>Unable to create organization</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="name">Organization Name</Label>
              <Input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
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
