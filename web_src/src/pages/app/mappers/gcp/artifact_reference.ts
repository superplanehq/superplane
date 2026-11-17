export function shortArtifactRef(uri?: string): string | undefined {
  if (!uri) {
    return undefined;
  }

  const normalized = uri.trim().replace(/^https?:\/\//, "");
  const slashIndex = normalized.indexOf("/");
  if (slashIndex < 0 || slashIndex === normalized.length - 1) {
    return undefined;
  }

  const path = normalized.slice(slashIndex + 1);
  const pathParts = path.split("/");
  if (pathParts.length < 3) {
    return path;
  }

  const { imagePath, ref, separator } = parseImageReference(pathParts.slice(2).join("/"));
  const imageName = imagePath.split("/").pop() || imagePath;
  if (!ref) {
    return imageName || path;
  }

  return `${imageName}${formatReferenceSuffix(ref, separator)}`;
}

function parseImageReference(imageWithRef: string): { imagePath: string; ref: string; separator: string } {
  const atIndex = imageWithRef.indexOf("@");
  if (atIndex >= 0) {
    return { imagePath: imageWithRef.slice(0, atIndex), ref: imageWithRef.slice(atIndex + 1), separator: "@" };
  }

  const lastSlash = imageWithRef.lastIndexOf("/");
  const lastColon = imageWithRef.lastIndexOf(":");
  if (lastColon > lastSlash) {
    return { imagePath: imageWithRef.slice(0, lastColon), ref: imageWithRef.slice(lastColon + 1), separator: ":" };
  }

  return { imagePath: imageWithRef, ref: "", separator: "" };
}

function formatReferenceSuffix(ref: string, separator: string): string {
  if (separator === "@" && ref.startsWith("sha256:")) {
    const digest = ref.slice("sha256:".length);
    return `@${digest.slice(0, 8)}`;
  }

  const compactRef = ref.length > 12 ? `${ref.slice(0, 12)}...` : ref;
  return `${separator}${compactRef}`;
}
