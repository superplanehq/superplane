const path = require('path');
const baseConfig = require('../../jest.config.browser');

module.exports = {
  ...baseConfig,
  displayName: __dirname.split(path.sep).slice(-2).join(path.posix.sep),
};
