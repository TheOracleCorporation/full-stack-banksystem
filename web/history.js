fetch("/api/history")
    .then(response => {
        if (!response.ok) {
            throw new Error("Ошибка получения истории");
        }

        return response.json();
    })
    .then(data => {

        const history = document.getElementById("history");

        if (data.length === 0) {
            history.textContent = "Операций пока нет";
            return;
        }

        history.innerHTML = "";

        data.forEach(transaction => {

            const item = document.createElement("div");

            let text = "";

            if (transaction.type === "deposit") {
                text = "+ " + transaction.amount + " ₽ — Пополнение";
            }

            if (transaction.type === "withdraw") {
                text = "- " + transaction.amount + " ₽ — Снятие";
            }

            if (transaction.type === "transfer") {

                if (transaction.from_account === null) {
                    text = "+ " + transaction.amount +
                        " ₽ — Перевод от " + transaction.other_user;
                } else {
                    text = "- " + transaction.amount +
                        " ₽ — Перевод " + transaction.other_user;
                }
            }

            item.textContent = text;

            item.style.padding = "12px 0";
            item.style.borderBottom = "1px solid #ddd";

            history.appendChild(item);
        });
    })
    .catch(error => {
        document.getElementById("history").textContent =
            error.message;
    });


document.getElementById("backButton").addEventListener("click", function() {
    window.location.href = "/account.html";
});