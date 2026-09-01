import { useAccount } from "@/contexts/useAccount";
import { useFactories } from "@/hooks/useFactoryData";
import { factoryListPath, factorySettingsSectionPath } from "@/pages/factories/lib/factoryPagePaths";
import { pickInitialFactory, readLastVisitedFactory } from "@/pages/factories/lib/lastVisitedFactory";
import { Navigate, useParams } from "react-router";

/** Old org settings URL. Send the user to workspace settings Notifications. */
export function Notifications() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  const { data: factories = [], isLoading } = useFactories(organizationId ?? "");

  if (!organizationId) {
    return <Navigate to="/" replace />;
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <p className="text-gray-500 dark:text-gray-400">Loading...</p>
      </div>
    );
  }

  const lastVisitedId = account?.id ? readLastVisitedFactory(account.id, organizationId) : null;
  const factory = pickInitialFactory(factories, lastVisitedId);
  if (!factory?.key) {
    return <Navigate to={factoryListPath(organizationId)} replace />;
  }

  return <Navigate to={factorySettingsSectionPath(organizationId, factory.key, "account", "notifications")} replace />;
}
