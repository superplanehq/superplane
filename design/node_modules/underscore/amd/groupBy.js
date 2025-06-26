define(['./_has', './_group'], function (_has, _group) {

  // Groups the object's values by a criterion. Pass either a string attribute
  // to group by, or a function that returns the criterion.
  var groupBy = _group(function(result, value, key) {
    if (_has(result, key)) result[key].push(value); else result[key] = [value];
  });

  return groupBy;

});
