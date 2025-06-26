var matcher = require('./matcher.js');
var find = require('./find.js');

// Convenience version of a common use case of `_.find`: getting the first
// object containing specific `key:value` pairs.
function findWhere(obj, attrs) {
  return find(obj, matcher(attrs));
}

module.exports = findWhere;
