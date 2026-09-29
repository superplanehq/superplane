export function machineTypeLabel(value: string): string {
  switch (value) {
    case "e1-tiny-amd64":
      return "Small x64";
    case "e1-tiny-arm64":
      return "Small ARM";
    case "e1-large-amd64":
      return "Large x64";
    case "e1-large-arm64":
      return "Large ARM";
    default:
      return value;
  }
}
