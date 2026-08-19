export function setSearchParamFlag(params: URLSearchParams, key: string, open: boolean): URLSearchParams {
  const next = new URLSearchParams(params);
  if (open) {
    next.set(key, "1");
    return next;
  }
  next.delete(key);
  return next;
}
