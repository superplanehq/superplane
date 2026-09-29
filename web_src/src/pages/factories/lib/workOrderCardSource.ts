import type { FactoriesWorkOrder } from "@/api-client";
import superplaneIcon from "@/assets/superplane.svg";

import {
  CREATED_MANUALLY,
  INTAKE_PRESENTATION,
  splitRunSourceForOrder,
  type SplitRunIntakeKind,
} from "../pages/work-order-split-run/splitRunSource";

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

/** Sentinel source filter value that matches tasks created by a person. */
export const MANUAL_FILTER_VALUE = "manual";

export function workOrderListSource(order: FactoriesWorkOrder): { id: string; label: string } {
  const source = splitRunSourceForOrder(order);
  if (source.kind === "manual") {
    return { id: MANUAL_FILTER_VALUE, label: CREATED_MANUALLY };
  }
  const intakeKind = intakeKindForPresentationName(source.name);
  return {
    id: intakeKind ?? source.name,
    label: source.name,
  };
}

function intakeKindForPresentationName(name: string): SplitRunIntakeKind | undefined {
  const kinds = Object.keys(INTAKE_PRESENTATION) as SplitRunIntakeKind[];
  return kinds.find((kind) => INTAKE_PRESENTATION[kind].name === name);
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
