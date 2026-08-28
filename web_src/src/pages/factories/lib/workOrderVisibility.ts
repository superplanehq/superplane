interface WorkOrderStateCarrier {
  state?: string;
}

/**
 * Intake work orders exist while an intake analyzes an item. Keep them in the
 * loaded API data for detail popups, but omit them from normal collections.
 */
export function visibleWorkOrdersForCollections<T extends WorkOrderStateCarrier>(workOrders: readonly T[]): T[] {
  return workOrders.filter((workOrder) => !isIntakeWorkOrder(workOrder));
}

export function isIntakeWorkOrder(workOrder: WorkOrderStateCarrier): boolean {
  return workOrder.state === "STATE_INTAKE";
}
