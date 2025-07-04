'use strict';

const ts = require('typescript');
const path = require('path');

function _interopNamespaceDefault(e) {
	const n = Object.create(null);
	if (e) {
		for (const k in e) {
			n[k] = e[k];
		}
	}
	n.default = e;
	return n;
}

const ts__namespace = /*#__PURE__*/_interopNamespaceDefault(ts);
const path__namespace = /*#__PURE__*/_interopNamespaceDefault(path);

const defaultPropFilter = (prop) => {
  return !prop.parent?.fileName.includes("node_modules");
};

function getTSConfigFile(tsconfigPath) {
  try {
    const basePath = path__namespace.dirname(tsconfigPath);
    const configFile = ts__namespace.readConfigFile(tsconfigPath, ts__namespace.sys.readFile);
    return ts__namespace.parseJsonConfigFileContent(
      configFile.config,
      ts__namespace.sys,
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
    jsx: ts__namespace.JsxEmit.React,
    module: ts__namespace.ModuleKind.CommonJS,
    target: ts__namespace.ScriptTarget.Latest
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

exports.getOptions = getOptions;
