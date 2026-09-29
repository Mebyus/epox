function getDocElem(id) {
    let element = document.getElementById(id);
    if (!element) {
        throw `element "${id}" not found`;
    }
    return element;
}

const EVENT_LOGIN_SUCCESS     = 1;
const EVENT_TASK_DONE_CLICK   = 2;
const EVENT_TASK_CANCEL_CLICK = 3;

class Reactor {
    constructor() {
        // maps event name to array of its handlers
        this.m = new Map();
    }

    emit = (name, data) => {
        let handlers = this.m.get(name);
        if (!handlers) {
            return;
        }

        for (let i = 0; i < handlers.length; i += 1) {
            const h = handlers[i];
            h(data);
        }
    }

    attach = (name, handler) => {
        let handlers = this.m.get(name);
        if (!handlers) {
            handlers = [];
            this.m.set(name, handlers);
        }
        handlers = handlers.push(handler);
    }
}

class API {
    // both login and password must be non-empty strings
    //
    // returns response object promise
    login = (login, password) => {
        return fetch("/ext/login", {
            method: "POST",
            body: JSON.stringify({
                login: login,
                password: password,
            }),
        })
        .then((response) => {
            return {
                ok: response.ok,
                body: null,
            }
        });
    }

    getActiveTasks = () => {
        return fetch("/ext/tasks/active", {
            method: "GET",
        })
        .then(async (response) => {
            return {
                ok: response.ok,
                body: await response.json(),
            }
        });
    }
}

class LoginForm {
    constructor(reactor, api) {
        this.api = api;
        this.reactor = reactor;

        this.de = getDocElem("login-form");
        this.de.addEventListener('submit', this.submit);
    }

    // handler for submit event
    submit = (e) => {
        e.preventDefault();

        const login = this.de.login.value.trim();
        const password = this.de.password.value.trim();

        if (!login || !password) {
            // TODO: show user error
            return;
        }

        this.api.login(login, password)
        .then((r) => {
            if (r.ok) {
                this.reactor.emit(EVENT_LOGIN_SUCCESS, null);
            } else {
                // TODO: display error
            }
        });
    }
}

class AuthView {
    constructor(reactor, api) {
        // view root DOM element
        this.de = getDocElem("view-auth");
        this.form = new LoginForm(reactor, api);
    }

    hide = () => {
        this.de.hidden = true;
    }

    show = () => {
        this.de.hidden = false;
    }
}

class TaskCard {
    constructor(reactor, task) {
        this.reactor = reactor;
        this.task = task;

        this.de = this.createElement(task);
    }

    // create and return DOM element for task card
    createElement = (task) => {
        let card = document.createElement("li");
        card.className = "task";
        card.tabIndex = 0;

        let taskCheck = document.createElement("label");
        taskCheck.className = "task-check";
        let checkbox = document.createElement("input");
        checkbox.type = "checkbox";
        taskCheck.appendChild(checkbox);
        let checkVisual = document.createElement("span");
        checkVisual.className = "task-check-visual";
        taskCheck.appendChild(checkVisual);
        card.appendChild(taskCheck);

        let taskBody = document.createElement("div");
        taskBody.className = "task-body";
        let taskLine = document.createElement("div");
        taskLine.className = "task-line";
        let taskTitle = document.createElement("span");
        taskTitle.className = "task-title";
        taskTitle.innerText = task.title;
        taskLine.appendChild(taskTitle);
        taskBody.appendChild(taskLine);
        card.appendChild(taskBody);

        let taskCancel = document.createElement("button");
        taskCancel.className = "task-cancel";
        taskCancel.type = "button";
        taskCancel.title = "Cancel";
        taskCancel.innerText = "✕";
        taskCancel.addEventListener("click", this.handleCancelClick);
        card.appendChild(taskCancel);

        return card;
    }

    handleCancelClick = (e) => {
        e.preventDefault();

        this.reactor.emit(EVENT_TASK_CANCEL_CLICK, this.task);
    }
}

class ActiveTaskPanel {
    constructor(reactor) {
        this.reactor = reactor;

        this.de = getDocElem("active-task-list");

        reactor.attach(EVENT_TASK_CANCEL_CLICK, this.handleTaskCancelClick);
    }

    addTaskCard = (task) => {
        const card = new TaskCard(this.reactor, task);
        this.de.appendChild(card.de);
    }

    handleTaskCancelClick = (data) => {
        const id = data.id;
        const title = data.title;
        console.log(`task ${title} (id=${id}) canceled`);
    }
}

class MainView {
    constructor(reactor, api) {
        this.api = api;

        // view root DOM element
        this.de = getDocElem("view-app");
        this.panels = {
            active: new ActiveTaskPanel(reactor),
        };
    }

    // load data from backend and render changes
    // accordingly
    load = () => {
        this.api.getActiveTasks()
        .then((r) => {
            const tasks = r.body;

            for (let i = 0; i < tasks.length; i += 1) {
                const t = tasks[i];
                this.panels.active.addTaskCard({
                    id: t.id,
                    title: t.title,
                });
            }
        });
    }

    hide = () => {
        this.de.hidden = true;
    }

    show = () => {
        this.de.hidden = false;
    }
}

class App {
    constructor() {
        this.api = new API();
        this.reactor = new Reactor();
        this.views = {
            auth: new AuthView(this.reactor, this.api),
            main: new MainView(this.reactor, this.api),
        };

        this.reactor.attach(EVENT_LOGIN_SUCCESS, this.handleLogin);
    }

    // switch view to specified one (by name)
    switchView = (name) => {
        switch (name) {
            case "auth":
                this.views.auth.show();
                this.views.main.hide();
                break;
            case "main":
                this.views.main.show();
                this.views.auth.hide();
                break;
            default:
                throw `unexpected "${name}" view`;
        }
    }

    handleLogin = (data) => {
        this.views.main.load();
        this.switchView("main");
    }
}

function main() {
    let app = new App();

    if (!window.user) {
        app.switchView("auth");
    } else {
        app.views.main.load();
        app.switchView("main");
    }
}

document.addEventListener('DOMContentLoaded', main);
