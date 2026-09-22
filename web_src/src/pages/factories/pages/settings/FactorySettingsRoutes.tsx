import { Route, Routes } from "react-router";

import { FactorySettingsLayout } from "./FactorySettingsLayout";
import { factorySettingsSectionRoutes } from "./factorySettingsSectionRoutes";

export function FactorySettingsRoutes() {
  return (
    <Routes>
      <Route element={<FactorySettingsLayout />}>{factorySettingsSectionRoutes}</Route>
    </Routes>
  );
}
