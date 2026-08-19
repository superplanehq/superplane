export function setSearchParamFlag(params: URLSearchParams, key: string, open: boolean): URLSearchParams {
  const isSet = params.get(key) === "1";
  if (open === isSet) {
    return params;
  }
  const next = new URLSearchParams(params);
  if (open) {
    next.set(key, "1");
    return next;
  }
  next.delete(key);
  return next;
}
