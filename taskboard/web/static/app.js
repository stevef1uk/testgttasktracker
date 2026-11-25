'use strict';
// TaskBoard client script: projects and tasks CRUD via the REST API.
(function () {
  "use strict";

  var state = { projects: [], tasks: [], projectId: null };

  function $(id) { return document.getElementById(id); }

  function api(path, opts) {
    var options = Object.assign({ headers: { "Content-Type": "application/json" } }, opts || {});
    return fetch(path, options).then(function (res) {
      return res.text().then(function (text) {
        var body = text ? JSON.parse(text) : null;
        if (!res.ok) {
          var msg = (body && body.error) ? body.error : ("HTTP " + res.status);
          throw new Error(msg);
        }
        return body;
      });
    });
  }

  function showError(err) {
    window.alert(err && err.message ? err.message : String(err));
  }

  function renderProjects() {
    var list = $("project-list");
    list.innerHTML = "";
    state.projects.forEach(function (p) {
      var li = document.createElement("li");
      var pick = document.createElement("button");
      pick.type = "button";
      pick.className = "select-project";
      pick.textContent = p.name;
      pick.addEventListener("click", function () {
        state.projectId = p.id;
        loadTasks();
      });
      var del = document.createElement("button");
      del.type = "button";
      del.className = "delete-project";
      del.textContent = "Delete";
      del.addEventListener("click", function () {
        if (!window.confirm("Delete project \"" + p.name + "\"?")) return;
        api("/api/projects/" + encodeURIComponent(p.id), { method: "DELETE" })
          .then(function () {
            if (state.projectId === p.id) {
              state.projectId = null;
              state.tasks = [];
            }
            return loadProjects();
          })
          .catch(showError);
      });
      li.appendChild(pick);
      li.appendChild(del);
      list.appendChild(li);
    });
  }

  var COLUMNS = ["todo", "doing", "done"];
  var NEXT_STATUS = { todo: "doing", doing: "done" };

  function patchTask(id, body) {
    return api("/api/tasks/" + encodeURIComponent(id), {
      method: "PATCH",
      body: JSON.stringify(body)
    }).then(loadTasks).catch(showError);
  }

  function makeButton(text, className, onClick) {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = className;
    btn.textContent = text;
    btn.addEventListener("click", onClick);
    return btn;
  }

  function renderTasks() {
    COLUMNS.forEach(function (s) { $("list-" + s).innerHTML = ""; });
    var fStatus = $("filter-status").value;
    var fPriority = $("filter-priority").value;
    state.tasks
      .filter(function (t) {
        return (!fStatus || t.status === fStatus) && (!fPriority || t.priority === fPriority);
      })
      .forEach(function (t) {
        var column = $("list-" + t.status);
        if (!column) return;
        var li = document.createElement("li");
        var label = document.createElement("span");
        label.textContent = t.title + " [" + t.priority + "] ";
        li.appendChild(label);
        if (NEXT_STATUS[t.status]) {
          li.appendChild(makeButton("Move forward", "move-forward", function () {
            patchTask(t.id, { status: NEXT_STATUS[t.status] });
          }));
        }
        li.appendChild(makeButton("Edit", "edit-task", function () {
          var newTitle = window.prompt("Task title", t.title);
          if (newTitle === null) return;
          newTitle = newTitle.trim();
          if (!newTitle) return;
          patchTask(t.id, { title: newTitle });
        }));
        li.appendChild(makeButton("Delete", "delete-task", function () {
          api("/api/tasks/" + encodeURIComponent(t.id), { method: "DELETE" })
            .then(loadTasks)
            .catch(showError);
        }));
        column.appendChild(li);
      });
  }

  function loadProjects() {
    return api("/api/projects").then(function (projects) {
      state.projects = projects || [];
      renderProjects();
    });
  }

  function loadTasks() {
    if (!state.projectId) {
      state.tasks = [];
      renderTasks();
      return Promise.resolve();
    }
    return api("/api/projects/" + encodeURIComponent(state.projectId) + "/tasks").then(function (tasks) {
      state.tasks = tasks || [];
      renderTasks();
    });
  }

  $("project-form").addEventListener("submit", function (e) {
    e.preventDefault();
    var name = $("project-name").value.trim();
    if (!name) return;
    api("/api/projects", { method: "POST", body: JSON.stringify({ name: name }) })
      .then(function () {
        $("project-name").value = "";
        return loadProjects();
      })
      .catch(showError);
  });

  $("task-form").addEventListener("submit", function (e) {
    e.preventDefault();
    if (!state.projectId) {
      window.alert("Select a project first");
      return;
    }
    var payload = {
      title: $("task-title").value.trim(),
      description: $("task-description").value,
      priority: $("task-priority").value
    };
    if (!payload.title) return;
    api("/api/projects/" + encodeURIComponent(state.projectId) + "/tasks", {
      method: "POST",
      body: JSON.stringify(payload)
    })
      .then(function () {
        $("task-title").value = "";
        $("task-description").value = "";
        return loadTasks();
      })
      .catch(showError);
  });

  $("filter-status").addEventListener("change", renderTasks);
  $("filter-priority").addEventListener("change", renderTasks);

  loadProjects().catch(showError);
})();