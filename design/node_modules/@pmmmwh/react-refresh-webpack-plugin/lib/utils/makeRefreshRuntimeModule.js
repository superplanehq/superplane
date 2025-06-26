/**
 * Makes a runtime module to intercept module execution for React Refresh.
 * This module creates an isolated __webpak_require__ function for each module
 * and injects a $Refresh$ object into it for use by react-refresh.
 * @param {import('webpack')} webpack The Webpack exports.
 * @returns {ReactRefreshRuntimeModule} The runtime module class.
 */
function makeRefreshRuntimeModule(webpack) {
  return class ReactRefreshRuntimeModule extends webpack.RuntimeModule {
    constructor() {
      // Second argument is the `stage` for this runtime module -
      // we'll use the same stage as Webpack's HMR runtime module for safety.
      super('react refresh', webpack.RuntimeModule.STAGE_BASIC);
    }

    /**
     * @returns {string} runtime code
     */
    generate() {
      const { runtimeTemplate } = this.compilation;
      const declareVar = runtimeTemplate.supportsConst() ? 'const' : 'var';
      return webpack.Template.asString([
        `${declareVar} setup = ${runtimeTemplate.basicFunction('moduleId', [
          `${declareVar} refresh = {`,
          webpack.Template.indent([
            `moduleId: moduleId,`,
            `register: ${runtimeTemplate.basicFunction('type, id', [
              `${declareVar} typeId = moduleId + " " + id;`,
              `refresh.runtime.register(type, typeId);`,
            ])},`,
            `signature: ${runtimeTemplate.returningFunction(
              'refresh.runtime.createSignatureFunctionForTransform()'
            )},`,
            `runtime: {`,
            webpack.Template.indent([
              `createSignatureFunctionForTransform: ${runtimeTemplate.returningFunction(
                runtimeTemplate.returningFunction('type', 'type')
              )},`,
              `register: ${runtimeTemplate.emptyFunction()}`,
            ]),
            `},`,
          ]),
          `};`,
          `return refresh;`,
        ])}`,
        '',
        `${webpack.RuntimeGlobals.interceptModuleExecution}.push(${runtimeTemplate.basicFunction(
          'options',
          [
            `${declareVar} originalFactory = options.factory;`,
            `options.factory = ${runtimeTemplate.basicFunction(
              ['moduleObject', 'moduleExports', 'webpackRequire'],
              [
                // Our require function delegates to the original require function
                `${declareVar} hotRequire = ${runtimeTemplate.returningFunction(
                  'webpackRequire(request)',
                  'request'
                )};`,
                // The propery descriptor factory below ensures that all properties but $Refresh$ are proxied through to the original require function
                `${declareVar} createPropertyDescriptor = ${runtimeTemplate.basicFunction('name', [
                  `return {`,
                  webpack.Template.indent([
                    `configurable: true,`,
                    `enumerable: true,`,
                    `get: ${runtimeTemplate.returningFunction('webpackRequire[name]')},`,
                    `set: ${runtimeTemplate.basicFunction('value', [
                      'webpackRequire[name] = value;',
                    ])},`,
                  ]),
                  `};`,
                ])};`,
                `for (${declareVar} name in webpackRequire) {`,
                webpack.Template.indent([
                  `if (Object.prototype.hasOwnProperty.call(webpackRequire, name) && name !== "$Refresh$") {`,
                  webpack.Template.indent([
                    `Object.defineProperty(hotRequire, name, createPropertyDescriptor(name));`,
                  ]),
                  `}`,
                ]),
                `}`,
                `hotRequire.$Refresh$ = setup(options.id);`,
                `originalFactory.call(this, moduleObject, moduleExports, hotRequire);`,
              ]
            )};`,
          ]
        )});`,
      ]);
    }
  };
}

module.exports = makeRefreshRuntimeModule;
