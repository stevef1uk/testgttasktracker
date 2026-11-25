(function () {
  'use strict';

  var NEXT_STATUS = { todo: 'doing', doing: 'done' };
  var state = { projectId: null };

  function $(id) {
    return document.getElementById(id);
  }

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) {
      node.className = className;
    }
    if (text !== undefined && text !== null) {
      node.textContent = String(text);
    }
    return node;
  }

  function api(method, url, body) {
    var opts = { method: method, headers: {} };
    if (body !== undefined) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    return fetch(url, opts).then(function (res) {
      if (res.status === 204) {
        return null;
      }
      return res.json().catch(function () {
        return {};
      }).then(function (data) {
        if (!res.ok) {
          throw new Error((data && data.error) || ('HTTP ' + res.status));
        }
        return data;
      });
    });
  }

  function reportError(err) {
    console.error(err);
    window.alert(err && err.message ? err.message : 'Request failed');
  }

  function loadProjects() {
    return api('GET', '/api/projects').then(function (projects) {
      var list = $('project-list');
      list.innerHTML = '';
      if (state.projectId !== null) {
        var stillExists = projects.some(function (p) {
          return p.id === state.projectId;
        });
        if (!stillExists) {
          state.projectId = null;
        }
      }
      projects.forEach(function (p) {
        var li = el('li', 'project-item');
        li.setAttribute('data-project-id', String(p.id));
        var select = el('button', 'project-select', p.name);
        select.type = 'button';
        select.addEventListener('click', function () {
          state.projectId = p.id;
          loadTasks().catch(reportError);
        });
        var remove = el('button', 'project-delete', 'Delete');
        remove.type = 'button';
        remove.addEventListener('click', function () {
          deleteProject(p.id).catch(reportError);
        });
        li.appendChild(select);
        li.appendChild(remove);
        if (p.id === state.projectId) {
          li.setAttribute('data-selected', 'true');
        }
        list.appendChild(li);
      });
      return loadTasks();
    });
  }

  function deleteProject(id) {
    return api('DELETE', '/api/projects/' + id).then(function () {
      if (state.projectId === id) {
        state.projectId = null;
      }
      return loadProjects();
    });
  }

  function loadTasks() {
    var list = $('task-list');
    list.innerHTML = '';
    if (state.projectId === null) {
      return Promise.resolve();
    }
    var params = new URLSearchParams();
    var status = $('filter-status').value;
    var priority = $('filter-priority').value;
    if (status) {
      params.set('status', status);
    }
    if (priority) {
      params.set('priority', priority);
    }
    var query = params.toString() ? '?' + params.toString() : '';
    return api('GET', '/api/projects/' + state.projectId + '/tasks' + query).then(function (tasks) {
      list.innerHTML = '';
      tasks.forEach(function (t) {
        list.appendChild(renderTask(t));
      });
    });
  }

  function renderTask(t) {
    var li = el('li', 'task-item');
    li.setAttribute('data-task-id', String(t.id));
    li.setAttribute('data-status', t.status);
    li.setAttribute('data-priority', t.priority);
    li.appendChild(el('strong', 'task-title', t.title));
    li.appendChild(el('span', 'task-status', t.status));
    li.appendChild(el('span', 'task-priority', t.priority));
    if (t.description) {
      li.appendChild(el('p', 'task-description', t.description));
    }

    var next = NEXT_STATUS[t.status];
    if (next) {
      var move = el('button', 'task-move', 'Move forward');
      move.type = 'button';
      move.addEventListener('click', function () {
        api('PATCH', '/api/tasks/' + t.id, { status: next })
          .then(loadTasks)
          .catch(reportError);
      });
      li.appendChild(move);
    }

    var remove = el('button', 'task-delete', 'Delete');
    remove.type = 'button';
    remove.addEventListener('click', function () {
      api('DELETE', '/api/tasks/' + t.id)
        .then(loadTasks)
        .catch(reportError);
    });
    li.appendChild(remove);
    return li;
  }

  function onProjectSubmit(event) {
    event.preventDefault();
    var input = $('project-name');
    var name = input.value.trim();
    if (!name) {
      return;
    }
    api('POST', '/api/projects', { name: name })
      .then(function (p) {
        input.value = '';
        state.projectId = p.id;
        return loadProjects();
      })
      .catch(reportError);
  }

  function onTaskSubmit(event) {
    event.preventDefault();
    if (state.projectId === null) {
      window.alert('Select a project first');
      return;
    }
    var title = $('task-title').value.trim();
    if (!title) {
      return;
    }
    var body = {
      title: title,
      description: $('task-description').value,
      status: $('task-status').value,
      priority: $('task-priority').value
    };
    api('POST', '/api/projects/' + state.projectId + '/tasks', body)
      .then(function () {
        $('task-title').value = '';
        $('task-description').value = '';
        return loadTasks();
      })
      .catch(reportError);
  }

  function init() {
    $('project-form').addEventListener('submit', onProjectSubmit);
    $('task-form').addEventListener('submit', onTaskSubmit);
    $('filter-status').addEventListener('change', function () {
      loadTasks().catch(reportError);
    });
    $('filter-priority').addEventListener('change', function () {
      loadTasks().catch(reportError);
    });
    loadProjects().catch(reportError);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();