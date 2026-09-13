# Мой банк

Учебное веб-приложение банка на Go: регистрация, вход, баланс, переводы между пользователями и история операций. Бэкенд — чистый `net/http` без фреймворков, фронтенд — HTML/CSS/JS без сборщиков.

## Возможности

- Регистрация и вход по email/паролю (хеширование пароля через `bcrypt`)
- Сессии на cookie (`HttpOnly`, `SameSite=Lax`)
- Пополнение и снятие средств
- Перевод другому пользователю по email
- История операций
- Просмотр профиля (имя, email, баланс)
- Выход из аккаунта
- Все операции с балансом атомарны (транзакции БД / условные `UPDATE`) — исключают гонки при параллельных запросах

## Стек

- **Backend:** Go, `net/http`, `database/sql`, [`github.com/mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3), [`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- **DB:** SQLite (`users.db`, создаётся автоматически при первом запуске)
- **Frontend:** HTML, CSS, vanilla JavaScript

## Структура проекта

```
.
├── main.go          # HTTP-сервер, роуты, работа с БД
├── go.mod / go.sum
├── users.db          # SQLite база (создаётся автоматически)
└── web/
    ├── index.html     # страница входа
    ├── script.js
    ├── register.html  # страница регистрации
    ├── register.js
    ├── account.html   # личный кабинет
    ├── account.js
    ├── profile.html   # профиль
    ├── profile.js
    ├── history.html   # история операций
    ├── history.js
    ├── modal.js       # модалки вместо alert()/prompt()
    └── style.css       # общие стили
```

Сервер поднимется на [http://localhost:8080](http://localhost:8080). База `users.db` и таблицы `users`/`transactions` создаются автоматически при первом запуске.

## API

Все запросы, кроме `/api/login` и `/api/register`, требуют cookie `session`.

| Метод | Путь             | Описание                        |
|-------|------------------|----------------------------------|
| POST  | `/api/register`  | Регистрация нового пользователя  |
| POST  | `/api/login`     | Вход, выставляет cookie сессии   |
| POST  | `/api/logout`    | Выход, удаляет сессию            |
| GET   | `/api/balance`   | Текущий баланс                   |
| GET   | `/api/profile`   | Имя, email, баланс                |
| POST  | `/api/deposit`   | Пополнение (`{ "amount": int }`) |
| POST  | `/api/withdraw`  | Снятие (`{ "amount": int }`)     |
| POST  | `/api/transfer`  | Перевод (`{ "email": string, "amount": int }`) |
| GET   | `/api/history`   | История операций                 |
