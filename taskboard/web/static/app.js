(function () {
  "use strict";

  var COLUMNS = ["todo", "doing", "done"];
  var NEXT = { todo: "doing", doing: "done" };

  var statusEl = document.getElementById("status");
  var errorEl = document.getElementById("error");
  var selectEl = document.getElementById("project-select");
  var projectForm = document.getElementById("project-form");
  var projectNameEl = document.getElementById("project-name");
  var taskForm = document.getElementById("task-form");
  var taskTitleEl = document.getElementById("task-title");
  var taskDescEl = document.getElementById("task-description");
  var taskPriorityEl = document.getElementById("task-priority");
  var statusFilterEl = document.getElementById("status-filter");
  var priorityFilterEl = document.getElementById("priority-filter");

  function showError(message) {
    errorEl.textContent = message;
    errorEl.hidden = false;
  }

  function clearError() {
    errorEl.textContent = "";
    errorEl.hidden = true;
  }

  function clearChildren(node) {
    while (node.firstChild) {
      node.removeChild(node.firstChild);
    }
  }

  function request(method, url, body) {
    var opts = { method: method, headers: { "Accept": "application/json" } };
    if (body !== undefined) {
      opts.headers["Content-Type"] = "application/json";
      opts.body = JSON.stringify(body);
    }
    return fetch(url, opts).then(function (res) {
      if (res.status === 204) {
        return null;
      }
      return res.json().catch(function () { return {}; }).then(function (data) {
        if (!res.ok) {
          throw new Error((data && data.error) || ("Request failed (HTTP " + res.status + ")"));
        }
        return data;
      });
    });
  }

  function currentProjectId() {
    return selectEl.value;
  }

  function renderProjects(projects) {
    var previous = selectEl.value;
    clearChildren(selectEl);
    var placeholder = document.createElement("option");
    placeholder.value = "";
    placeholder.textContent = "Select a project";
    selectEl.appendChild(placeholder);
    (projects || []).forEach(function (p) {
      var opt = document.createElement("option");
      opt.value = String(p.id);
      opt.textContent = p.name;
      selectEl.appendChild(opt);
    });
    selectEl.value = previous;
  }

  function loadProjects() {
    return request("GET", "/api/projects")
      .then(function (projects) {
        renderProjects(projects);
        statusEl.textContent = "Ready";
        clearError();
      })
      .catch(function (err) {
        statusEl.textContent = "Unavailable";
        showError(err.message);
      });
  }

  function buildCard(task) {
    var card = document.createElement("div");
    card.className = "card";
    card.setAttribute("data-task-id", String(task.id));

    var title = document.createElement("h4");
    title.textContent = task.title;
    card.appendChild(title);

    if (task.description) {
      var desc = document.createElement("p");
      desc.textContent = task.description;
      card.appendChild(desc);
    }

    var pill = document.createElement("span");
    pill.className = "pill pill-" + task.priority;
    pill.textContent = task.priority;
    card.appendChild(pill);

    var actions = document.createElement("div");
    actions.className = "actions";

    var move = document.createElement("button");
    move.type = "button";
    move.className = "secondary";
    move.textContent = "Move forward";
    if (!NEXT[task.status]) {
      move.disabled = true;
    }
    move.addEventListener("click", function () {
      patchTask(task.id, { status: NEXT[task.status] });
    });
    actions.appendChild(move);

    var edit = document.createElement("button");
    edit.type = "button";
    edit.className = "secondary";
    edit.textContent = "Edit";
    edit.addEventListener("click", function () {
      var title = window.prompt("Task title", task.title);
      if (title === null) {
        return;
      }
      var priority = window.prompt("Priority (low, medium, high)", task.priority);
      if (priority === null) {
        return;
      }
      patchTask(task.id, { title: title, priority: priority });
    });
    actions.appendChild(edit);

    var del = document.createElement("button");
    del.type = "button";
    del.className = "danger";
    del.textContent = "Delete";
    del.addEventListener("click", function () {
      deleteTask(task.id);
    });
    actions.appendChild(del);

    card.appendChild(actions);
    return card;
  }

  function renderTasks(tasks) {
    COLUMNS.forEach(function (status) {
      var list = document.querySelector('[data-column="' + status + '"]');
      clearChildren(list);
    });
    (tasks || []).forEach(function (task) {
      var list = document.querySelector('[data-column="' + task.status + '"]');
      if (list) {
        list.appendChild(buildCard(task));
      }
    });
  }

  function loadTasks() {
    var projectId = currentProjectId();
    if (!projectId) {
      COLUMNS.forEach(function (status) {
        clearChildren(document.querySelector('[data-column="' + status + '"]'));
      });
      return Promise.resolve();
    }
    var params = new URLSearchParams();
    if (statusFilterEl.value !== "all") {
      params.set("status", statusFilterEl.value);
    }
    if (priorityFilterEl.value !== "all") {
      params.set("priority", priorityFilterEl.value);
    }
    var query = params.toString();
    var url = "/api/projects/" + encodeURIComponent(projectId) + "/tasks" + (query ? "?" + query : "");
    return request("GET", url)
      .then(function (tasks) {
        renderTasks(tasks);
        clearError();
      })
      .catch(function (err) {
        showError(err.message);
      });
  }

  function patchTask(id, body) {
    return request("PATCH", "/api/tasks/" + id, body)
      .then(function () { return loadTasks(); })
      .catch(function (err) { showError(err.message); });
  }

  function deleteTask(id) {
    return request("DELETE", "/api/tasks/" + id)
      .then(function () { return loadTasks(); })
      .catch(function (err) { showError(err.message); });
  }

  projectForm.addEventListener("submit", function (event) {
    event.preventDefault();
    clearError();
    var name = projectNameEl.value.trim();
    if (name === "") {
      showError("Project name is required");
      return;
    }
    request("POST", "/api/projects", { name: name })
      .then(function () {
        projectForm.reset();
        return loadProjects();
      })
      .catch(function (err) { showError(err.message); });
  });

  taskForm.addEventListener("submit", function (event) {
    event.preventDefault();
    clearError();
    var projectId = currentProjectId();
    if (!projectId) {
      showError("Select a project before adding a task");
      return;
    }
    var title = taskTitleEl.value.trim();
    if (title === "") {
      showError("Task title is required");
      return;
    }
    request("POST", "/api/projects/" + encodeURIComponent(projectId) + "/tasks", {
      title: title,
      description: taskDescEl.value,
      priority: taskPriorityEl.value
    })
      .then(function () {
        taskForm.reset();
        return loadTasks();
      })
      .catch(function (err) { showError(err.message); });
  });

  selectEl.addEventListener("change", function () { loadTasks(); });
  statusFilterEl.addEventListener("change", function () { loadTasks(); });
  priorityFilterEl.addEventListener("change", function () { loadTasks(); });

  loadProjects().then(loadTasks);
})();