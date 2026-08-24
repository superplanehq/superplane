import { Navigate } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId } from "../lib/factoryPagePaths";

/** Sends the workspace index to the line board (or the empty lines list). */
export function FactoryHomeRedirect() {
  const { organizationId, factoryKey, factory } = useFactoriesLayout();
  return (
    <Navigate
      to={{ pathname: factoryHomePath(organizationId, factoryKey, firstFactoryLineId(factory)), search: "" }}
      replace
    />
  );
}
