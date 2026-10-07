/** Factory right sidebar: component label first, optional gray node name after ·. */
export function resolveFactorySidebarHeading({
  componentLabel,
  nodeName,
}: {
  componentLabel?: string;
  nodeName?: string;
}): { primary: string; secondary: string | null } {
  const label = componentLabel?.trim();
  const name = nodeName?.trim();
  if (label && name) {
    return { primary: label, secondary: name };
  }
  if (label) {
    return { primary: label, secondary: null };
  }
  return { primary: name || "Component", secondary: null };
}
