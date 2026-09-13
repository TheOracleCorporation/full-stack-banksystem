function loadBalance() {
    fetch("/api/balance")
        .then(response => {
            if (!response.ok) {
                throw new Error("Ошибка получения баланса");
            }

            return response.json();
        })
        .then(data => {
            document.getElementById("balance").textContent =
                data.balance + " ₽";
        })
        .catch(error => {
            document.getElementById("balance").textContent =
                "Ошибка";
        });
}


loadBalance();


document.getElementById("depositButton").addEventListener("click", async function() {

    const amount = await showPrompt("Пополнение баланса", {
        label: "Введите сумму пополнения",
        type: "number",
        placeholder: "0"
    });

    if (amount === null) {
        return;
    }

    const number = Number(amount);

    if (number <= 0 || !Number.isInteger(number)) {
        await showAlert("Введите целое число больше нуля");
        return;
    }

    fetch("/api/deposit", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            amount: number
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error("Ошибка пополнения");
        }

        return response.json();
    })
    .then(async data => {
        document.getElementById("balance").textContent =
            data.balance + " ₽";

        await showAlert("Баланс успешно пополнен!");
    })
    .catch(async error => {
        await showAlert(error.message);
    });
});

document.getElementById("withdrawButton").addEventListener("click", async function() {

    const amount = await showPrompt("Снятие средств", {
        label: "Введите сумму снятия",
        type: "number",
        placeholder: "0"
    });

    if (amount === null) {
        return;
    }

    const number = Number(amount);

    if (number <= 0 || !Number.isInteger(number)) {
        await showAlert("Введите целое число больше нуля");
        return;
    }

    fetch("/api/withdraw", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            amount: number
        })
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(message => {
                throw new Error(message);
            });
        }

        return response.json();
    })
    .then(async data => {
        document.getElementById("balance").textContent =
            data.balance + " ₽";

        await showAlert("Средства успешно сняты!");
    })
    .catch(async error => {
        await showAlert(error.message);
    });
});

document.getElementById("transferButton").addEventListener("click", async function() {

    const values = await openModal({
        title: "Перевод пользователю",
        fields: [
            { label: "Email получателя", type: "email", placeholder: "you@example.com" },
            { label: "Сумма перевода", type: "number", placeholder: "0" }
        ],
        confirmText: "Перевести"
    });

    if (values === null) {
        return;
    }

    const [email, amount] = values;
    const number = Number(amount);

    if (number <= 0 || !Number.isInteger(number)) {
        await showAlert("Введите целое число больше нуля");
        return;
    }

    fetch("/api/transfer", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            email: email,
            amount: number
        })
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(message => {
                throw new Error(message);
            });
        }

        return response.json();
    })
    .then(async data => {
        document.getElementById("balance").textContent =
            data.balance + " ₽";

        await showAlert("Перевод успешно выполнен!");
    })
    .catch(async error => {
        await showAlert(error.message);
    });
});

document.getElementById("historyButton").addEventListener("click", function() {
    window.location.href = "/history.html";
});

document.getElementById("profileButton").addEventListener("click", function() {
    window.location.href = "/profile.html";
});

document.getElementById("logoutButton").addEventListener("click", function() {
    fetch("/api/logout", {
        method: "POST"
    })
    .finally(function() {
        window.location.href = "/";
    });
});