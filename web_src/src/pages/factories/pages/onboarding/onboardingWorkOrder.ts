interface CreatedWorkOrder {
  id?: string | null;
  number?: number | string | null;
}

export async function createAndDispatchInitialWorkOrder(args: {
  title: string;
  description: string;
  lineName: string;
  createWorkOrder: (input: { title: string; description: string }) => Promise<CreatedWorkOrder>;
  dispatchWorkOrder: (input: { orderId: string; lineName: string }) => Promise<unknown>;
}): Promise<CreatedWorkOrder> {
  const order = await args.createWorkOrder({
    title: args.title,
    description: args.description,
  });
  if (!order.id) {
    throw new Error("Created work order has no ID");
  }

  await args.dispatchWorkOrder({ orderId: order.id, lineName: args.lineName });
  return order;
}
