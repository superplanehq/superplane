define(['./_setup', './identity', './isArray', './_isArrayLike', './values', './isString', './map'], function (_setup, identity, isArray, _isArrayLike, values, isString, map) {

  // Safely create a real, live array from anything iterable.
  var reStrSymbol = /[^\ud800-\udfff]|[\ud800-\udbff][\udc00-\udfff]|[\ud800-\udfff]/g;
  function toArray(obj) {
    if (!obj) return [];
    if (isArray(obj)) return _setup.slice.call(obj);
    if (isString(obj)) {
      // Keep surrogate pair characters together.
      return obj.match(reStrSymbol);
    }
    if (_isArrayLike(obj)) return map(obj, identity);
    return values(obj);
  }

  return toArray;

});
