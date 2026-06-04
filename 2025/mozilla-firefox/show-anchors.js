'use strict';

function prepend(old) {
   const a = document.createElement('a');
   const s = 'id' in old ? old.id : old.name;
   a.textContent = '#' + s;
   /* must assume site is using <base> and "hash" */
   a.href = location.pathname + '#' + s;
   old.prepend(a);
}

document.querySelectorAll('[id], [name]').forEach(prepend);
