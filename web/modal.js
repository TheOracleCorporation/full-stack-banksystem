// Простая модалка в стиле проекта — замена браузерным prompt()/alert().
// openModal возвращает Promise: массив значений полей при подтверждении,
// или null при отмене/клике мимо/Esc.
function openModal({ title, message, fields = [], confirmText = "ОК", cancelText = "Отмена", onlyOk = false }) {
    return new Promise(function(resolve) {

        const overlay = document.createElement("div");
        overlay.className = "modal-overlay";

        const box = document.createElement("div");
        box.className = "modal-box";

        const form = document.createElement("form");

        if (title) {
            const titleEl = document.createElement("h2");
            titleEl.className = "modal-title";
            titleEl.textContent = title;
            form.appendChild(titleEl);
        }

        if (message) {
            const messageEl = document.createElement("p");
            messageEl.className = "modal-message";
            messageEl.textContent = message;
            form.appendChild(messageEl);
        }

        const errorEl = document.createElement("p");
        errorEl.className = "modal-error";
        errorEl.style.display = "none";

        const inputs = [];

        fields.forEach(function(field, index) {
            const wrap = document.createElement("div");
            wrap.className = "field";

            const label = document.createElement("label");
            label.textContent = field.label;
            const inputId = "modal-field-" + index;
            label.setAttribute("for", inputId);

            const input = document.createElement("input");
            input.id = inputId;
            input.type = field.type || "text";
            if (field.placeholder) input.placeholder = field.placeholder;
            if (field.min !== undefined) input.min = field.min;

            wrap.appendChild(label);
            wrap.appendChild(input);
            form.appendChild(wrap);
            inputs.push(input);
        });

        if (fields.length > 0) {
            form.insertBefore(errorEl, form.lastChild.nextSibling);
        } else {
            form.appendChild(errorEl);
        }

        const actions = document.createElement("div");
        actions.className = "modal-actions";

        function close(result) {
            document.removeEventListener("keydown", onKeydown);
            document.body.removeChild(overlay);
            resolve(result);
        }

        function onKeydown(event) {
            if (event.key === "Escape" && !onlyOk) {
                close(null);
            }
        }

        if (!onlyOk) {
            const cancelBtn = document.createElement("button");
            cancelBtn.type = "button";
            cancelBtn.className = "secondary";
            cancelBtn.textContent = cancelText;
            cancelBtn.addEventListener("click", function() {
                close(null);
            });
            actions.appendChild(cancelBtn);
        }

        const okBtn = document.createElement("button");
        okBtn.type = "submit";
        okBtn.textContent = confirmText;
        actions.appendChild(okBtn);

        form.appendChild(actions);

        form.addEventListener("submit", function(event) {
            event.preventDefault();

            if (fields.length > 0) {
                for (let i = 0; i < fields.length; i++) {
                    if (fields[i].required !== false && inputs[i].value.trim() === "") {
                        errorEl.textContent = "Заполните все поля";
                        errorEl.style.display = "block";
                        return;
                    }
                }
            }

            close(inputs.map(function(input) { return input.value; }));
        });

        box.appendChild(form);
        overlay.appendChild(box);
        document.body.appendChild(overlay);

        overlay.addEventListener("mousedown", function(event) {
            if (event.target === overlay && !onlyOk) {
                close(null);
            }
        });

        document.addEventListener("keydown", onKeydown);

        if (inputs[0]) {
            inputs[0].focus();
        } else {
            okBtn.focus();
        }
    });
}

// Замена alert(): одна кнопка "ОК", закрывается по клику/Enter/Esc.
function showAlert(message) {
    return openModal({ message: message, confirmText: "ОК", onlyOk: true });
}

// Замена prompt() с одним полем. Возвращает строку или null при отмене.
function showPrompt(title, options) {
    options = options || {};
    return openModal({
        title: title,
        fields: [{ label: options.label || title, type: options.type || "text", placeholder: options.placeholder }],
        confirmText: options.confirmText || "ОК"
    }).then(function(values) {
        return values ? values[0] : null;
    });
}