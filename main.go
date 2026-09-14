package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var sessions = make(map[string]int)
var sessionsMutex sync.Mutex

func register(db *sql.DB) {
	fmt.Println("Введите ваш никнейм:")
	var name string
	fmt.Scanln(&name)

	fmt.Println("Введите ваш email:")
	var email string
	fmt.Scanln(&email)

	fmt.Println("Введите пароль:")
	var password string
	fmt.Scanln(&password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	_, err = db.Exec(
		"INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
		name,
		email,
		string(hash),
	)

	if err != nil {
		fmt.Println("Ошибка регистрации:", err)
		return
	}

	fmt.Println("Регистрация успешна!")

}

func login(db *sql.DB) (int, error) {
	fmt.Println("Введите ваш email:")
	var email string
	fmt.Scanln(&email)

	fmt.Println("Введите пароль:")
	var password string
	fmt.Scanln(&password)

	var name string
	var hashedPassword string
	var userID int

	err := db.QueryRow("SELECT id, password, name FROM users WHERE email = ?", email).Scan(&userID, &hashedPassword, &name)
	if err == sql.ErrNoRows {
		fmt.Println("Ошибка: Пользователь не найден!")
		return 0, err
	} else if err != nil {
		fmt.Println("Ошибка при поиске в базе:", err)
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		fmt.Println("Ошибка: Неверный пароль!")
		return 0, err
	}

	fmt.Println("Вход выполнен успешно! Добро пожаловать,", name)

	return userID, nil
}

func userMenu(db *sql.DB, userID int) {
	for {
		fmt.Println("\n===== ЛИЧНЫЙ КАБИНЕТ =====")
		fmt.Println("1. Посмотреть баланс")
		fmt.Println("2. Пополнить баланс")
		fmt.Println("3. Снять средства")
		fmt.Println("4. Перевести пользователю")
		fmt.Println("5. Мои данные")
		fmt.Println("6. История операций")
		fmt.Println("7. Выйти")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			showBalance(db, userID)
		case 2:
			deposit(db, userID)
		case 3:
			withdraw(db, userID)
		case 4:
			transfer(db, userID)
		case 5:
			showMe(db, userID)
		case 6:
			showHistory(db, userID)
		case 7:
			fmt.Println("Вы вышли из аккаунта.")
			return
		default:
			fmt.Println("Ошибка: такого пункта нет.")
		}
	}
}

func showBalance(db *sql.DB, userID int) {
	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		fmt.Println("Ошибка при получении баланса:", err)
		return
	}
	fmt.Println("Ваш баланс:", balance)
}

func deposit(db *sql.DB, userID int) {
	fmt.Println("Введите сумму пополнения:")
	var amount int
	fmt.Scanln(&amount)

	if amount <= 0 {
		fmt.Println("Ошибка: сумма должна быть положительной")
		return
	}

	_, err := db.Exec("UPDATE users SET balance = balance + ? WHERE id = ?", amount, userID)
	if err != nil {
		fmt.Println("Ошибка пополнения:", err)
		return
	}

	db.Exec(
		"INSERT INTO transactions (from_account, to_account, amount, type, created_at) VALUES (?, ?, ?, ?, ?)",
		nil, userID, amount, "deposit", time.Now(),
	)

	fmt.Println("Баланс пополнен на", amount)
}

func depositHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		var data struct {
			Amount int `json:"amount"`
		}

		err = json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		if data.Amount <= 0 {
			http.Error(w, "Сумма должна быть больше нуля", http.StatusBadRequest)
			return
		}

		_, err = db.Exec(
			`INSERT INTO transactions
			(from_account, to_account, amount, type, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			nil,
			userID,
			data.Amount,
			"deposit",
			time.Now(),
		)

		if err != nil {
			http.Error(w, "Ошибка записи транзакции", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(
			"UPDATE users SET balance = balance + ? WHERE id = ?",
			data.Amount,
			userID,
		)

		if err != nil {
			http.Error(w, "Ошибка пополнения", http.StatusInternalServerError)
			return
		}

		var newBalance int

		err = db.QueryRow(
			"SELECT balance FROM users WHERE id = ?",
			userID,
		).Scan(&newBalance)

		if err != nil {
			http.Error(w, "Ошибка получения баланса", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]int{
			"balance": newBalance,
		})
	}
}

func withdraw(db *sql.DB, userID int) {
	fmt.Println("Введите сумму снятия:")
	var amount int
	fmt.Scanln(&amount)

	if amount <= 0 {
		fmt.Println("Ошибка: сумма должна быть положительной")
		return
	}

	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil {
		fmt.Println("Ошибка при проверке баланса:", err)
		return
	}
	if balance < amount {
		fmt.Println("Ошибка: недостаточно средств")
		return
	}

	_, err = db.Exec("UPDATE users SET balance = balance - ? WHERE id = ?", amount, userID)
	if err != nil {
		fmt.Println("Ошибка снятия:", err)
		return
	}

	db.Exec(
		"INSERT INTO transactions (from_account, to_account, amount, type, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, nil, amount, "withdraw", time.Now(),
	)

	fmt.Println("Снято", amount, "с баланса")
}

func withdrawHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		var data struct {
			Amount int `json:"amount"`
		}

		err = json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		if data.Amount <= 0 {
			http.Error(w, "Сумма должна быть больше нуля", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Атомарное списание: строка обновится, только если средств
		// достаточно на момент выполнения запроса. Это закрывает гонку,
		// при которой два одновременных запроса на снятие могли пройти
		// проверку баланса до того, как любой из них его обновит.
		result, err := tx.Exec(
			"UPDATE users SET balance = balance - ? WHERE id = ? AND balance >= ?",
			data.Amount,
			userID,
			data.Amount,
		)

		if err != nil {
			http.Error(w, "Ошибка снятия", http.StatusInternalServerError)
			return
		}

		affected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Ошибка снятия", http.StatusInternalServerError)
			return
		}

		if affected == 0 {
			http.Error(w, "Недостаточно средств", http.StatusBadRequest)
			return
		}

		_, err = tx.Exec(
			`INSERT INTO transactions
			(from_account, to_account, amount, type, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			userID,
			nil,
			data.Amount,
			"withdraw",
			time.Now(),
		)

		if err != nil {
			http.Error(w, "Ошибка записи транзакции", http.StatusInternalServerError)
			return
		}

		var newBalance int

		err = tx.QueryRow(
			"SELECT balance FROM users WHERE id = ?",
			userID,
		).Scan(&newBalance)

		if err != nil {
			http.Error(w, "Ошибка получения баланса", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Ошибка сохранения операции", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]int{
			"balance": newBalance,
		})
	}
}

func transfer(db *sql.DB, userID int) {
	fmt.Println("Введите email получателя:")
	var toEmail string
	fmt.Scanln(&toEmail)

	fmt.Println("Введите сумму перевода:")
	var amount int
	fmt.Scanln(&amount)

	if amount <= 0 {
		fmt.Println("Ошибка: сумма должна быть положительной")
		return
	}

	var toID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", toEmail).Scan(&toID)
	if err != nil {
		fmt.Println("Ошибка: получатель не найден")
		return
	}
	if toID == userID {
		fmt.Println("Ошибка: нельзя перевести самому себе")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Ошибка базы данных:", err)
		return
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT balance FROM users WHERE id = ?", userID).Scan(&balance)
	if err != nil || balance < amount {
		fmt.Println("Ошибка: недостаточно средств")
		return
	}

	if _, err := tx.Exec("UPDATE users SET balance = balance - ? WHERE id = ?", amount, userID); err != nil {
		fmt.Println("Ошибка перевода:", err)
		return
	}
	if _, err := tx.Exec("UPDATE users SET balance = balance + ? WHERE id = ?", amount, toID); err != nil {
		fmt.Println("Ошибка перевода:", err)
		return
	}
	if _, err := tx.Exec(
		"INSERT INTO transactions (from_account, to_account, amount, type, created_at) VALUES (?, ?, ?, ?, ?)",
		userID, toID, amount, "transfer", time.Now(),
	); err != nil {
		fmt.Println("Ошибка записи истории:", err)
		return
	}

	if err := tx.Commit(); err != nil {
		fmt.Println("Ошибка сохранения перевода:", err)
		return
	}

	fmt.Println("Перевод выполнен успешно!")
}

func transferHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Получаем сессию отправителя
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		// Получаем данные перевода
		var data struct {
			Email  string `json:"email"`
			Amount int    `json:"amount"`
		}

		err = json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		// Проверяем сумму
		if data.Amount <= 0 {
			http.Error(w, "Сумма должна быть больше нуля", http.StatusBadRequest)
			return
		}

		// Ищем получателя
		var recipientID int

		err = db.QueryRow(
			"SELECT id FROM users WHERE email = ?",
			data.Email,
		).Scan(&recipientID)

		if err != nil {
			http.Error(w, "Получатель не найден", http.StatusNotFound)
			return
		}

		// Нельзя переводить самому себе
		if recipientID == userID {
			http.Error(w, "Нельзя перевести деньги самому себе", http.StatusBadRequest)
			return
		}

		// Вся операция выполняется в одной транзакции: либо и списание,
		// и зачисление, и запись в историю проходят вместе, либо
		// откатывается всё (раньше сбой на середине операции мог списать
		// деньги у отправителя, но не зачислить их получателю).
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Атомарное списание с проверкой баланса в самом запросе —
		// закрывает гонку, при которой два одновременных перевода могли
		// оба пройти проверку баланса до того, как любой из них его
		// обновит, и увести баланс в минус.
		result, err := tx.Exec(
			"UPDATE users SET balance = balance - ? WHERE id = ? AND balance >= ?",
			data.Amount,
			userID,
			data.Amount,
		)

		if err != nil {
			http.Error(w, "Ошибка списания", http.StatusInternalServerError)
			return
		}

		affected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Ошибка списания", http.StatusInternalServerError)
			return
		}

		if affected == 0 {
			http.Error(w, "Недостаточно средств", http.StatusBadRequest)
			return
		}

		// Зачисляем деньги получателю
		_, err = tx.Exec(
			"UPDATE users SET balance = balance + ? WHERE id = ?",
			data.Amount,
			recipientID,
		)

		if err != nil {
			http.Error(w, "Ошибка зачисления", http.StatusInternalServerError)
			return
		}

		// Записываем перевод в историю
		_, err = tx.Exec(
			`INSERT INTO transactions
			(from_account, to_account, amount, type, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			userID,
			recipientID,
			data.Amount,
			"transfer",
			time.Now(),
		)

		if err != nil {
			http.Error(w, "Ошибка записи транзакции", http.StatusInternalServerError)
			return
		}

		// Получаем новый баланс
		var newBalance int

		err = tx.QueryRow(
			"SELECT balance FROM users WHERE id = ?",
			userID,
		).Scan(&newBalance)

		if err != nil {
			http.Error(w, "Ошибка получения баланса", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Ошибка сохранения перевода", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]int{
			"balance": newBalance,
		})
	}
}

func showMe(db *sql.DB, userID int) {
	var name, email string
	var balance int
	err := db.QueryRow("SELECT name, email, balance FROM users WHERE id = ?", userID).
		Scan(&name, &email, &balance)
	if err != nil {
		fmt.Println("Ошибка при получении данных:", err)
		return
	}
	fmt.Println("Имя:", name)
	fmt.Println("Email:", email)
	fmt.Println("Баланс:", balance)
}

func showHistory(db *sql.DB, userID int) {
	rows, err := db.Query(
		`SELECT id, from_account, to_account, amount, type, created_at 
         FROM transactions 
         WHERE from_account = ? OR to_account = ? 
         ORDER BY created_at DESC`,
		userID, userID,
	)
	if err != nil {
		fmt.Println("Ошибка при получении истории:", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n--- История операций ---")
	for rows.Next() {
		var id int
		var from, to sql.NullInt64
		var amount int
		var txType string
		var createdAt time.Time

		rows.Scan(&id, &from, &to, &amount, &txType, &createdAt)
		fmt.Printf("[%s] %s: %d (from=%v to=%v)\n", createdAt.Format("2006-01-02 15:04"), txType, amount, from, to)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func balanceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		var balance int

		err = db.QueryRow(
			"SELECT balance FROM users WHERE id = ?",
			userID,
		).Scan(&balance)

		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]int{
			"balance": balance,
		})
	}
}

func webLoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var data struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		var userID int
		var hashedPassword string

		err = db.QueryRow(
			"SELECT id, password FROM users WHERE email = ?",
			data.Email,
		).Scan(&userID, &hashedPassword)

		if err != nil {
			http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword(
			[]byte(hashedPassword),
			[]byte(data.Password),
		)

		if err != nil {
			http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
			return
		}

		tokenBytes := make([]byte, 32)

		_, err = rand.Read(tokenBytes)
		if err != nil {
			http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
			return
		}

		token := hex.EncodeToString(tokenBytes)

		sessionsMutex.Lock()
		sessions[token] = userID
		sessionsMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			// Secure: true, // включить, когда сайт работает по HTTPS
		})

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Вход выполнен успешно",
		})
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		// Куки нет — считаем, что пользователь уже вышел.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Вы вышли из аккаунта",
		})
		return
	}

	sessionsMutex.Lock()
	delete(sessions, cookie.Value)
	sessionsMutex.Unlock()

	// Затираем куку в браузере: то же имя/путь, пустое значение,
	// истёкший срок действия.
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Вы вышли из аккаунта",
	})
}

func webRegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Неверный запрос", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(data.Name) == "" || strings.TrimSpace(data.Email) == "" {
			http.Error(w, "Имя и email обязательны", http.StatusBadRequest)
			return
		}

		if len(data.Password) < 6 {
			http.Error(w, "Пароль должен быть не короче 6 символов", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(data.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			http.Error(w, "Ошибка создания пароля", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(
			"INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
			data.Name,
			data.Email,
			string(hashedPassword),
		)

		if err != nil {
			fmt.Println("Ошибка регистрации:", err)
			http.Error(w, "Ошибка регистрации", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Регистрация прошла успешно",
		})
	}
}

func historyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT
    			transactions.amount,
    			transactions.type,
    			transactions.created_at,
    			transactions.from_account,
    			transactions.to_account,
    			COALESCE(users.name, '')
			FROM transactions
			LEFT JOIN users
				ON users.id =
					CASE
						WHEN transactions.from_account = ? THEN transactions.to_account
						ELSE transactions.from_account
					END
			WHERE transactions.from_account = ?
			   OR transactions.to_account = ?
			ORDER BY transactions.created_at DESC
		`, userID, userID, userID)

		if err != nil {
			fmt.Println("Ошибка чтения истории:", err)
			http.Error(w, "Ошибка чтения истории", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type Transaction struct {
			Amount      int    `json:"amount"`
			Type        string `json:"type"`
			CreatedAt   string `json:"created_at"`
			FromAccount *int   `json:"from_account"`
			ToAccount   *int   `json:"to_account"`
			OtherUser   string `json:"other_user"`
		}

		var transactions []Transaction

		for rows.Next() {
			var transaction Transaction

			err = rows.Scan(
				&transaction.Amount,
				&transaction.Type,
				&transaction.CreatedAt,
				&transaction.FromAccount,
				&transaction.ToAccount,
				&transaction.OtherUser,
			)

			if err != nil {
				fmt.Println("Ошибка получения истории:", err)
				http.Error(w, "Ошибка получения истории", http.StatusInternalServerError)
				return
			}

			transactions = append(transactions, transaction)
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(transactions)
	}
}

func profileHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Вы не авторизованы", http.StatusUnauthorized)
			return
		}

		sessionsMutex.Lock()
		userID, exists := sessions[cookie.Value]
		sessionsMutex.Unlock()

		if !exists {
			http.Error(w, "Сессия недействительна", http.StatusUnauthorized)
			return
		}

		var name string
		var email string
		var balance int

		err = db.QueryRow(
			"SELECT name, email, balance FROM users WHERE id = ?",
			userID,
		).Scan(&name, &email, &balance)

		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    name,
			"email":   email,
			"balance": balance,
		})
	}
}

func main() {

	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return
	}

	defer db.Close()

	fmt.Println("База данных подключена!")

	_, err = db.Exec(`
    	CREATE TABLE IF NOT EXISTS users (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	name TEXT NOT NULL,
   		email TEXT NOT NULL UNIQUE,
    	password TEXT NOT NULL,
		balance INTEGER NOT NULL DEFAULT 0
		)
	`)

	if err != nil {
		fmt.Println("Ошибка создания таблицы:", err)
		return
	}

	_, err = db.Exec(`
    	CREATE TABLE IF NOT EXISTS transactions (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	from_account INTEGER,
    	to_account INTEGER,
    	amount INTEGER NOT NULL,
    	type TEXT NOT NULL,
    	created_at DATETIME NOT NULL,
    	FOREIGN KEY (from_account) REFERENCES users(id),
    	FOREIGN KEY (to_account) REFERENCES users(id)
    	)
	`)
	if err != nil {
		fmt.Println("Ошибка создания таблицы transactions:", err)
		return
	}

	http.HandleFunc("/api/login", webLoginHandler(db))
	http.HandleFunc("/api/register", webRegisterHandler(db))
	http.Handle("/api/balance", balanceHandler(db))
	http.HandleFunc("/api/deposit", depositHandler(db))
	http.HandleFunc("/api/withdraw", withdrawHandler(db))
	http.HandleFunc("/api/transfer", transferHandler(db))
	http.HandleFunc("/api/history", historyHandler(db))
	http.HandleFunc("/api/profile", profileHandler(db))
	http.HandleFunc("/api/logout", logoutHandler)

	http.Handle("/", http.FileServer(http.Dir("web")))

	fmt.Println("Сервер запущен на http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Println("Ошибка сервера:", err)
	}
}


