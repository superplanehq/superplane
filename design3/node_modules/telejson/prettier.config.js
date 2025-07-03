const base = require('@storybook/linter-config/prettier.config');

module.exports = Object.assign({}, base, {
  overrides: [
    {
      files: '*.html',
      options: { parser: 'babel' },
    },
  ],
});
