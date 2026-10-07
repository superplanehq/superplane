export function shouldRedirectFactoryAppCanvas({
  appId,
  canvasLoading,
  canvasError,
  belongsToFactory,
  hasCanvas,
}: {
  appId: string;
  canvasLoading: boolean;
  canvasError: unknown;
  belongsToFactory: boolean;
  hasCanvas: boolean;
}) {
  if (!appId) {
    return true;
  }
  return !canvasLoading && Boolean(canvasError || (hasCanvas && !belongsToFactory));
}
