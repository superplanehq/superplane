define(['./_binarySearch', './_cb', './_lessEqual'], function (_binarySearch, _cb, _lessEqual) {

  // Use an iteratee to figure out the greatest index at which an object should be
  // inserted so as to maintain order. Uses binary search.
  function sortedLastIndex(array, obj, iteratee, context) {
    return _binarySearch(array, obj, _cb(iteratee, context), _lessEqual);
  }

  return sortedLastIndex;

});
