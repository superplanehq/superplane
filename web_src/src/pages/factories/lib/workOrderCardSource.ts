import type { FactoriesWorkOrder } from "@/api-client";
import superplaneIcon from "@/assets/superplane.svg";

import { CREATED_MANUALLY, splitRunSourceForOrder } from "../pages/work-order-split-run/splitRunSource";

export interface WorkOrderCardSource {
  name: string;
  iconSrc: string;
  iconAlt: string;
  ticket?: { label: string; href: string };
  creatorName?: string;
}

export function workOrderCardSource(order: FactoriesWorkOrder): WorkOrderCardSource | null {
  const source = splitRunSourceForOrder(order);
  if (source.kind === "manual") {
    const creatorName = order.createdBy?.user?.name?.trim();
    return {
      name: CREATED_MANUALLY,
      iconSrc: superplaneIcon,
      iconAlt: "SuperPlane",
      ...(creatorName ? { creatorName } : {}),
    };
  }
  if (source.kind !== "intake") {
    return null;
  }
  return {
    name: source.name,
    iconSrc: source.iconSrc,
    iconAlt: source.iconAlt,
    ticket: source.ticket,
  };
}

export function workOrderCardSourceLabel(source: WorkOrderCardSource): string {
  if (source.name === CREATED_MANUALLY) {
    const creatorName = source.creatorName?.trim();
    if (creatorName) {
      return `${CREATED_MANUALLY} by ${creatorName}`;
    }
    return CREATED_MANUALLY;
  }
  const ticketLabel = source.ticket?.label?.trim();
  if (!ticketLabel) {
    return source.name;
  }
  return `${source.name} ${ticketLabel}`;
}
