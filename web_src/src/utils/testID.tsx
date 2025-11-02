export function toTestId(name: string) {
  return name.toLowerCase().replace(/\s+/g, "-");
}
