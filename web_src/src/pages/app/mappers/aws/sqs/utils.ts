/**
 * Extracts the queue name from an SQS queue URL.
 *
 * @param queueUrl - The SQS queue URL (e.g., "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue")
 * @returns The queue name or undefined if the URL is invalid
 */
export function getQueueNameFromUrl(queueUrl?: string): string | undefined {
  if (!queueUrl) {
    return undefined;
  }

  const trimmed = queueUrl.trim();
  if (!trimmed) {
    return undefined;
  }

  try {
    const url = new URL(trimmed);
    const pathParts = url.pathname.split("/").filter((part) => part.length > 0);
    if (pathParts.length === 0) {
      return undefined;
    }

    // The queue name is the last part of the path
    return pathParts[pathParts.length - 1];
  } catch {
    // If URL parsing fails, try to extract from the string directly
    const parts = trimmed.split("/");
    if (parts.length === 0) {
      return undefined;
    }

    const lastPart = parts[parts.length - 1];
    return lastPart && lastPart.trim().length > 0 ? lastPart.trim() : undefined;
  }
}

// Export for TypeScript module resolution
export {};
