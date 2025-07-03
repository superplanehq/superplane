import * as ts from 'typescript';
import * as path from 'path';

const defaultPropFilter = (prop) => {
  return !prop.parent?.fileName.includes("node_modules");
};

function getTSConfigFile(tsconfigPath) {
  try {
    const basePath = path.dirname(tsconfigPath);
    const configFile = ts.readConfigFile(tsconfigPath, ts.sys.readFile);
    return ts.parseJsonConfigFileContent(
      configFile.config,
      ts.sys,
      basePath,
      {},
      tsconfigPath
    );
  } catch (error) {
    return {};
  }
}

function getOptions(options) {
  const {
    tsconfigPath = "./tsconfig.json",
    compilerOptions: userCompilerOptions,
    setDisplayName = true,
    typePropName = "type",
    propFilter = defaultPropFilter,
    ...docgenOptions
  } = options;
  let compilerOptions = {
    jsx: ts.JsxEmit.React,
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.Latest
  };
  if (userCompilerOptions) {
    compilerOptions = {
      ...compilerOptions,
      ...userCompilerOptions
    };
  } else {
    const { options: tsOptions } = getTSConfigFile(tsconfigPath);
    compilerOptions = { ...compilerOptions, ...tsOptions };
  }
  return {
    docgenOptions: {
      propFilter,
      ...docgenOptions
    },
    generateOptions: {
      setDisplayName,
      typePropName
    },
    compilerOptions
  };
}

export { getOptions };
