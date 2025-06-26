Object.defineProperty(exports, '__esModule', { value: true });

var _isNaN = require('./isNaN.js');

// Internal `extremum` return value adapter for `_.min` and `_.max`.
// Ensures that a number is returned even if no element of the
// collection maps to a numeric value.
function decideNumeric(fallback) {
  return function(result, iterResult) {
    return _isNaN(+iterResult) ? fallback : result;
  }
}

exports.decideNumeric = decideNumeric;
