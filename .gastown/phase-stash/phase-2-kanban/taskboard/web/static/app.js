'use strict';
// TaskBoard client script: loads projects from the REST API and renders them into the page.
(function () {
  var list = document.getElementById('project-list');
  if (!list) {
    return;
  }
  fetch('/api/projects')
    .then(function (res) {
      if (!res.ok) {
        throw new Error('HTTP ' + res.status);
      }
      return res.json();
    })
    .then(function (projects) {
      list.innerHTML = '';
      projects.forEach(function (p) {
        var li = document.createElement('li');
        li.textContent = p.name;
        list.appendChild(li);
      });
    })
    .catch(function () {
      list.textContent = 'Failed to load projects.';
    });
})();