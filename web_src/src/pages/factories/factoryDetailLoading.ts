export function getFactoryDetailLoadingState(input: {
  factoryLoading: boolean;
  ordersLoading: boolean;
  ordersFetching: boolean;
  workOrderCount: number;
  appsLoading: boolean;
  appsFetching: boolean;
  factoryAppCount: number;
}) {
  const hasWorkOrders = input.workOrderCount > 0;
  const hasFactoryApps = input.factoryAppCount > 0;

  return {
    isLoading: input.factoryLoading || (input.ordersLoading && !hasWorkOrders),
    isOrdersLoading: input.ordersLoading || (input.ordersFetching && !hasWorkOrders),
    isAppsLoading: input.appsLoading || (input.appsFetching && !hasFactoryApps),
  };
}
