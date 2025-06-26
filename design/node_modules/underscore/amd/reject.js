define(['./_cb', './filter', './negate'], function (_cb, filter, negate) {

  // Return all the elements for which a truth test fails.
  function reject(obj, predicate, context) {
    return filter(obj, negate(_cb(predicate)), context);
  }

  return reject;

});
