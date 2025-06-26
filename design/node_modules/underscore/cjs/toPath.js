var isArray = require('./isArray.js');
var underscore = require('./underscore.js');

// Normalize a (deep) property `path` to array.
// Like `_.iteratee`, this function can be customized.
function toPath(path) {
  return isArray(path) ? path : [path];
}
underscore.toPath = toPath;

module.exports = toPath;
