'use strict';

const vite = require('vite');
const path = require('path');
const glob = require('glob-promise');

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

const path__namespace = /*#__PURE__*/_interopNamespaceDefault(path);

const getUtils = async (config) => {
  const docGen = await import('react-docgen-typescript');
  const { default: ts } = await import('typescript');
  const { generateDocgenCodeBlock } = await import('./chunks/generate.cjs');
  const { getOptions } = await import('./chunks/options.cjs');
  const { docgenOptions, compilerOptions, generateOptions } = getOptions(config);
  const docGenParser = docGen.withCompilerOptions(
    compilerOptions,
    docgenOptions
  );
  const { exclude = ["**/**.stories.tsx"], include = ["**/**.tsx"] } = docgenOptions;
  const filter = vite.createFilter(include, exclude);
  const files = include.map(
    (filePath) => glob.sync(
      path__namespace.isAbsolute(filePath) ? filePath : path__namespace.join(process.cwd(), filePath)
    )
  ).reduce((carry, files2) => carry.concat(files2), []);
  const tsProgram = ts.createProgram(files, compilerOptions);
  const result = {
    docGenParser,
    filter,
    generateOptions,
    generateDocgenCodeBlock,
    tsProgram
  };
  return result;
};
function reactDocgenTypescript(config = {}) {
  const utilsPromise = getUtils(config);
  return {
    name: "vite:react-docgen-typescript",
    async transform(src, id) {
      try {
        const {
          filter,
          docGenParser,
          generateOptions,
          generateDocgenCodeBlock,
          tsProgram
        } = await utilsPromise;
        if (!filter(id)) {
          return;
        }
        const componentDocs = docGenParser.parseWithProgramProvider(
          id,
          () => tsProgram
        );
        if (!componentDocs.length) {
          return null;
        }
        return generateDocgenCodeBlock({
          filename: id,
          source: src,
          componentDocs,
          ...generateOptions
        });
      } catch (e) {
        return src;
      }
    }
  };
}

module.exports = reactDocgenTypescript;
