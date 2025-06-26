var _binarySearch = require('./_binarySearch.js');
var _cb = require('./_cb.js');
var _lessEqual = require('./_lessEqual.js');

// Use an iteratee to figure out the greatest index at which an object should be
// inserted so as to maintain order. Uses binary search.
function sortedLastIndex(array, obj, iteratee, context) {
  return _binarySearch(array, obj, _cb(iteratee, context), _lessEqual);
}

module.exports = sortedLastIndex;
