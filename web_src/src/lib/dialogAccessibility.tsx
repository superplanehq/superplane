import * as React from "react";

function getReactDisplayName(childType: unknown): string | undefined {
  if (typeof childType === "string") {
    return childType;
  }

  if (childType && (typeof childType === "function" || typeof childType === "object") && "displayName" in childType) {
    return typeof childType.displayName === "string" ? childType.displayName : undefined;
  }

  return undefined;
}

export function hasDialogChildOfType(children: React.ReactNode, matchingComponents: readonly unknown[]): boolean {
  const matchingDisplayNames = new Set(
    matchingComponents.map(getReactDisplayName).filter((value): value is string => !!value),
  );

  return React.Children.toArray(children).some((child) => {
    if (!React.isValidElement<{ children?: React.ReactNode }>(child)) {
      return false;
    }

    const childType = child.type;
    const displayName = getReactDisplayName(childType);

    if (matchingComponents.includes(childType) || (!!displayName && matchingDisplayNames.has(displayName))) {
      return true;
    }

    return hasDialogChildOfType(child.props.children, matchingComponents);
  });
}
