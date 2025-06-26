var _setup = require('./_setup.js');
var restArguments = require('./restArguments.js');

// Internal function to wrap `Array.prototype` methods that return a
// new value so they can be directly invoked as standalone functions.
// Works for `concat`, `slice` and `join`.
function wrapArrayAccessor(name) {
  var method = _setup.ArrayProto[name];
  return restArguments(function(array, args) {
    return array == null ? array : method.apply(array, args);
  });
}

module.exports = wrapArrayAccessor;
