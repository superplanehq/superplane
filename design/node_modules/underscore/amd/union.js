define(['./_flatten', './restArguments', './uniq'], function (_flatten, restArguments, uniq) {

  // Produce an array that contains the union: each distinct element from all of
  // the passed-in arrays.
  var union = restArguments(function(arrays) {
    return uniq(_flatten(arrays, true, true));
  });

  return union;

});
