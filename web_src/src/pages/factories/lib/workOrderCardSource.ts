import type { FactoriesWorkOrder } from "@/api-client";

import { splitRunSourceForOrder } from "../pages/work-order-split-run/splitRunSource";

export interface WorkOrderCardSource {
  name: string;
  iconSrc: string;
  iconAlt: string;
  ticket?: { label: string; href: string };
}

export function workOrderCardSource(order: FactoriesWorkOrder): WorkOrderCardSource | null {
  const source = splitRunSourceForOrder(order);
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
  const ticketLabel = source.ticket?.label?.trim();
  if (!ticketLabel) {
    return source.name;
  }
  return `${source.name} ${ticketLabel}`;
}
