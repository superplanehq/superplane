define(['./property', './map'], function (property, map) {

  // Convenience version of a common use case of `_.map`: fetching a property.
  function pluck(obj, key) {
    return map(obj, property(key));
  }

  return pluck;

});
