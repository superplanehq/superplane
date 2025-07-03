// babel-preset-env: `false` means ESM modules, `undefined` means CJS modules

module.exports = {
  presets: [
    ["@babel/preset-env", { targets: { node: "current" } }],
    '@babel/preset-typescript',
  ],
};
