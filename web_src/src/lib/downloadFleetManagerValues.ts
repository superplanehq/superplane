export function downloadFleetManagerValues(values: string) {
  const url = URL.createObjectURL(new Blob([values], { type: "application/yaml;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = "values.yaml";
  link.click();
  URL.revokeObjectURL(url);
}
