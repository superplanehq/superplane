/**
 * System notification messages are user-role messages with a well-known prefix.
 * They are rendered as centered grey text in the chat (not as bubbles).
 * The agent sees them in conversation history and can react to state changes.
 *
 * Format: "@@system: <message text>"
 *
 * TODO: Replace with a dedicated message type/role when the DB schema supports it.
 * This would involve adding a `type` column to agent_session_messages
 * (e.g. "message", "system_notification") and updating the proto/API.
 */

export const SYSTEM_PREFIX = "@@system: ";

export function isSystemNotification(content: string): boolean {
  return content.startsWith(SYSTEM_PREFIX);
}

export function formatSystemNotification(content: string): string {
  return content.slice(SYSTEM_PREFIX.length);
}

export function createSystemMessage(text: string): string {
  return `${SYSTEM_PREFIX}${text}`;
}
