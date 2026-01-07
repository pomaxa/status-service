# Status Incident Service

Внутренний сервис мониторинга статусов систем и отслеживания инцидентов.

## Возможности

- **Управление системами** - добавление проектов/сервисов с описанием, URL и ответственным
- **Зависимости** - отслеживание компонентов каждой системы (БД, Redis, API и т.д.)
- **Статусы-светофор** - green (работает), yellow (деградация), red (недоступен)
- **Ручное обновление** - изменение статуса с комментарием
- **Heartbeat мониторинг** - автоматическая проверка URL каждую минуту
- **История изменений** - полный лог всех изменений статусов
- **Аналитика** - uptime/SLA, количество инцидентов, MTTR

## Технологии

- **Backend:** Go + chi router
- **Database:** SQLite (WAL mode)
- **Frontend:** HTML templates + vanilla JS
- **Архитектура:** DDD (Domain-Driven Design)

## Запуск

```bash
# Сборка
go build -o status-incident .

# Запуск (порт 8080)
./status-incident
```

Сервис будет доступен по адресу http://localhost:8080

## Структура проекта

```
├── main.go                     # Точка входа
├── internal/
│   ├── domain/                 # Бизнес-логика (entities, value objects)
│   ├── application/            # Use cases (сервисы)
│   ├── infrastructure/         # SQLite репозитории, HTTP checker
│   └── interfaces/             # HTTP handlers, background workers
├── templates/                  # HTML шаблоны
└── static/                     # CSS стили
```

## Web-интерфейс

| Страница | URL | Описание |
|----------|-----|----------|
| Dashboard | `/` | Обзор всех систем |
| System | `/systems/{id}` | Детали системы и зависимостей |
| Admin | `/admin` | Управление системами |
| Logs | `/logs` | История изменений |
| Analytics | `/analytics` | Статистика и SLA |

## REST API

### Системы

```bash
# Список систем
GET /api/systems

# Создать систему
POST /api/systems
{"name": "API", "description": "Main API", "url": "https://api.example.com", "owner": "Backend Team"}

# Получить систему
GET /api/systems/{id}

# Обновить систему
PUT /api/systems/{id}
{"name": "API", "description": "Updated", "url": "https://api.example.com", "owner": "Backend Team"}

# Удалить систему
DELETE /api/systems/{id}

# Изменить статус
POST /api/systems/{id}/status
{"status": "yellow", "message": "Degraded performance"}
```

### Зависимости

```bash
# Список зависимостей системы
GET /api/systems/{id}/dependencies

# Добавить зависимость
POST /api/systems/{id}/dependencies
{"name": "PostgreSQL", "description": "Main database"}

# Обновить зависимость
PUT /api/dependencies/{id}

# Удалить зависимость
DELETE /api/dependencies/{id}

# Изменить статус зависимости
POST /api/dependencies/{id}/status
{"status": "red", "message": "Connection lost"}

# Настроить heartbeat
POST /api/dependencies/{id}/heartbeat
{"url": "https://api.example.com/health", "interval": 60}

# Отключить heartbeat
DELETE /api/dependencies/{id}/heartbeat

# Принудительная проверка
POST /api/dependencies/{id}/check
```

### Аналитика

```bash
# Общая аналитика
GET /api/analytics?period=24h

# Аналитика системы
GET /api/systems/{id}/analytics?period=7d

# Все логи
GET /api/logs?limit=100
```

## Heartbeat логика

- Проверка URL каждую минуту
- HTTP 200 = успех, иначе = ошибка
- 1 ошибка подряд → статус **yellow**
- 3 ошибки подряд → статус **red**
- Успешная проверка → сброс счетчика, статус **green**

## Лицензия

MIT
