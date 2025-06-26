define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var sort = _unmethodize(_setup.ArrayProto.sort);

	return sort;

});
