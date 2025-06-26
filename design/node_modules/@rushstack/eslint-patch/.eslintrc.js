// This is a workaround for https://github.com/eslint/eslint/issues/3458
require('@rushstack/heft-node-rig/profiles/default/includes/eslint/patch/modern-module-resolution');
// This is a workaround for https://github.com/microsoft/rushstack/issues/3021
require('@rushstack/heft-node-rig/profiles/default/includes/eslint/patch/custom-config-package-names');

module.exports = {
  extends: [
    '@rushstack/heft-node-rig/profiles/default/includes/eslint/profile/node',
    '@rushstack/heft-node-rig/profiles/default/includes/eslint/mixins/friendly-locals',
    '@rushstack/heft-node-rig/profiles/default/includes/eslint/mixins/tsdoc'
  ],
  parserOptions: { tsconfigRootDir: __dirname },
  plugins: ['eslint-plugin-header'],
  overrides: [
    {
      files: ['*.ts', '*.tsx'],
      rules: {
        'header/header': [
          'warn',
          'line',
          [
            ' Copyright (c) Microsoft Corporation. All rights reserved. Licensed under the MIT license.',
            ' See LICENSE in the project root for license information.'
          ]
        ]
      }
    }
  ]
};
