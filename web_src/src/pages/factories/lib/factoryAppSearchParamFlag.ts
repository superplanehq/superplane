/** Query flags that belong to factory Configure, not the canvas run page. */
const FACTORY_CONFIGURE_SEARCH_KEYS = ["configure", "edit", "blocks", "agent", "yaml", "agentPrompt"] as const;

/** Drop Configure chrome flags. Keep run/from/line/order context. */
export function leaveFactoryConfigureSearchParams(current: URLSearchParams): URLSearchParams {
  let changed = false;
  for (const key of FACTORY_CONFIGURE_SEARCH_KEYS) {
    if (current.has(key)) {
      changed = true;
      break;
    }
  }
  if (!changed) {
    return current;
  }
  const next = new URLSearchParams(current);
  for (const key of FACTORY_CONFIGURE_SEARCH_KEYS) {
    next.delete(key);
  }
  return next;
}

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
