// Suppresses the realtime "staging_updated" echo a tab receives for its own
// staging writes. Staging writes persist server-side and broadcast to every tab
// (including the originating one); without this guard the originating tab would
// refetch staged caches and could clobber newer, still in-flight local edits.
//
// State is module-scoped and therefore per-tab: other tabs run their own module
// instance with an empty registry, so they still react to the broadcast.

const PENDING_ECHO_TTL_MS = 5000;

// Expiry timestamps per `${canvasId}:${versionId}`. Each entry represents one
// in-flight local staging write awaiting its broadcast echo. A TTL guards
// against writes that fail (or whose echo never arrives) leaking forever.
const pendingEchoExpiries = new Map<string, number[]>();

function echoKey(canvasId: string, versionId: string): string {
  return `${canvasId}:${versionId}`;
}

function dropExpired(expiries: number[], now: number): void {
  while (expiries.length > 0 && expiries[0] <= now) {
    expiries.shift();
  }
}

// Registers a staging write issued by this tab so its broadcast echo can be
// ignored. Call right before issuing the request, since the server may broadcast
// before the request's HTTP response resolves.
export function registerLocalStagingWrite(canvasId?: string, versionId?: string): void {
  if (!canvasId || !versionId) {
    return;
  }

  const key = echoKey(canvasId, versionId);
  const now = Date.now();
  const expiries = pendingEchoExpiries.get(key) ?? [];
  dropExpired(expiries, now);
  expiries.push(now + PENDING_ECHO_TTL_MS);
  pendingEchoExpiries.set(key, expiries);
}

// Reports whether the incoming staging_updated event matches a local write from
// this tab, consuming the registration when it does so a later genuine remote
// event for the same version is not suppressed.
export function consumeLocalStagingWrite(canvasId?: string, versionId?: string): boolean {
  if (!canvasId || !versionId) {
    return false;
  }

  const key = echoKey(canvasId, versionId);
  const expiries = pendingEchoExpiries.get(key);
  if (!expiries) {
    return false;
  }

  dropExpired(expiries, Date.now());
  if (expiries.length === 0) {
    pendingEchoExpiries.delete(key);
    return false;
  }

  expiries.shift();
  if (expiries.length === 0) {
    pendingEchoExpiries.delete(key);
  }
  return true;
}
