var get = require('./get.js');
var noop = require('./noop.js');

// Generates a function for a given object that returns a given property.
function propertyOf(obj) {
  if (obj == null) return noop;
  return function(path) {
    return get(obj, path);
  };
}

module.exports = propertyOf;
