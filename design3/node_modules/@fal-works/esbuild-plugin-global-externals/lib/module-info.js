export const normalizeModuleInfo = (value) => {
  const {
    type = "esm",
    varName,
    namedExports = null,
    defaultExport = true,
  } = typeof value === "string" ? { varName: value } : value;
  return { type, varName, namedExports, defaultExport };
};
