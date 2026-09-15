const loadedSrcs = new Set<string>();
const LOADED_SRC_LIMIT = 200;

export function clearWorkspaceMarkdownImageLoadCache() {
  loadedSrcs.clear();
}

export function workspaceMarkdownImageIsLoaded(src: string | undefined): boolean {
  return Boolean(src && loadedSrcs.has(src));
}

export function rememberWorkspaceMarkdownImageLoad(src: string) {
  loadedSrcs.delete(src);
  loadedSrcs.add(src);
  if (loadedSrcs.size <= LOADED_SRC_LIMIT) {
    return;
  }
  const oldest = loadedSrcs.keys().next().value;
  if (oldest !== undefined) {
    loadedSrcs.delete(oldest);
  }
}

export function forgetWorkspaceMarkdownImageLoad(src: string) {
  loadedSrcs.delete(src);
}
