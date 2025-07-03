import type { NormalizedModuleInfo } from "./module-info";

const createCjsContents = (variableName: string) =>
  `module.exports = ${variableName};`;

const createEsmContents = (
  variableName: string,
  namedExports: readonly string[] | null,
  defaultExport: boolean
) => {
  const codeElements = defaultExport ? [`export default ${variableName};`] : [];

  if (namedExports && namedExports.length) {
    const exportNames = [...new Set(namedExports)].join(", ");
    codeElements.push(`const { ${exportNames} } = ${variableName};`);
    codeElements.push(`export { ${exportNames} };`);
  }

  return codeElements.join("\n");
};

/**
 * Creates value for `OnLoadResult.contents`.
 */
export const createContents = (moduleInfo: NormalizedModuleInfo): string => {
  const { type, varName, namedExports, defaultExport } = moduleInfo;

  switch (type) {
    case "esm":
      return createEsmContents(varName, namedExports, defaultExport);
    case "cjs":
      return createCjsContents(varName);
  }
};
