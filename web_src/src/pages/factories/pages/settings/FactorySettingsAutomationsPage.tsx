import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import { AutomationsPage } from "../AutomationsPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

/**
 * Settings home for the factory automations list. Reuses the old
 * Automations page and supplies the workspace layout context that page
 * still reads.
 */
export function FactorySettingsAutomationsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();

  return (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId,
        factoryId,
        factoryKey: factory.key ?? "",
        factory,
        factories: [factory],
        openCreateWorkOrder: () => undefined,
      }}
    >
      <AutomationsPage layout="settings" />
    </FactoriesLayoutContext.Provider>
  );
}
