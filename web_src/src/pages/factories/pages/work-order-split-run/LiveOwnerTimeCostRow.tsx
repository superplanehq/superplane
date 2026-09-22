import { type ComponentProps } from "react";

import { OwnerTimeCostRow } from "../work-order-popup-redesign/popupShared";
import { useLiveHeaderSpendOverlay } from "./liveHeaderSpendContext";

export function LiveOwnerTimeCostRow(props: ComponentProps<typeof OwnerTimeCostRow>) {
  const liveSpend = useLiveHeaderSpendOverlay();
  return <OwnerTimeCostRow {...props} liveSpend={liveSpend} />;
}
