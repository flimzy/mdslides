if ( typeof($global.$) === 'undefined' ) {
    try {
        $global.$ = require('jquery');
    } catch(e) {
        throw("Cannot find global jQuery object. Did you load jQuery?");
    }
}
