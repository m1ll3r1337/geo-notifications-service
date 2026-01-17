# Geo-Notifications Service

## Оглавление
- [Запуск](#запуск)
- [Миграции](#миграции)
- [Тестирование с ngrok](#тестирование-с-ngrok)
- [API](#api)
  - [Публичные эндпоинты](#публичные-эндпоинты)
  - [Защищённые эндпоинты](#защищённые-эндпоинты)
- [Архитектура вебхуков](#архитектура-вебхуков)  

## Запуск
1. Создайте `.env` (см. `.env.example`).
2. Запустите: `docker compose up --build`.
3. API доступен на `http://localhost:8080`.

## Миграции
Миграции применяются автоматически в `docker-compose.yml` (сервис `migrate`).

## Тестирование с ngrok (для тестирования вебхуков)
1. Запустите: `ngrok http 9090`.
2. Скопируйте URL в `.env`: `GEO_WORKERS_WEBHOOK_URL=https://abc123.ngrok.io/webhook`.

## API

Полную коллекцию см. в `Geo Notifications API.postman_collection.json`.

### Публичные эндпоинты

#### POST /api/v1/location/check
Проверка координат на ближайшие инциденты.
```bash
curl -X POST http://localhost:8080/api/v1/location/check \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user1",
    "location": {"lat": 55.7558, "lon": 37.6173},
    "limit": 10
  }'
```
**Ответ (200):**
```json
{
  "count": 1,
  "incidents": [
    {
      "incident_id": 1,
      "distance_meters": 150.5,
      "title": "Flooding in downtown",
      "description": "Major flood warning",
      "center": {"lat": 55.7558, "lon": 37.6173},
      "radius": 500,
      "created_at": "2026-01-01T12:00:00Z",
      "updated_at": "2026-01-01T12:00:00Z"
    }
  ]
}
```

#### GET /api/v1/health
Health-check сервиса.
```bash
curl http://localhost:8080/api/v1/health
```
**Ответ (200):**
```json
{
  "status": "ok",
  "timestamp": "2026-01-01T12:00:00Z",
  "dependencies": {
    "postgres": {"status": "ok"},
    "redis_cache": {"status": "ok"},
    "redis_queue": {"status": "ok"}
  }
}
```

### Защищённые эндпоинты (API-key: `secret`)

#### POST /api/v1/incidents
Создание нового инцидента.
```bash
curl -X POST http://localhost:8080/api/v1/incidents \
  -H "Content-Type: application/json" \
  -H "X-API-Key: secret" \
  -d '{
    "title": "Flooding in downtown",
    "description": "Major flood warning in the city center",
    "center": {"lat": 55.7558, "lon": 37.6173},
    "radius": 500
  }'
```
**Ответ (201):**
```json
{
  "id": 1,
  "title": "Flooding in downtown",
  "description": "Major flood warning in the city center",
  "center": {"lat": 55.7558, "lon": 37.6173},
  "radius": 500,
  "active": true,
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-01-01T12:00:00Z"
}
```

#### GET /api/v1/incidents/{id}
Получение инцидента по ID.
```bash
curl -H "X-API-Key: secret" http://localhost:8080/api/v1/incidents/1
```
**Ответ (200):**
```json
{
  "id": 1,
  "title": "Flooding in downtown",
  "description": "Major flood warning in the city center",
  "center": {"lat": 55.7558, "lon": 37.6173},
  "radius": 500,
  "active": true,
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-01-01T12:00:00Z"
}
```

#### GET /api/v1/incidents
Список инцидентов с фильтрами.
```bash
curl -H "X-API-Key: secret" "http://localhost:8080/api/v1/incidents?limit=50&offset=0&active_only=true"
```
**Ответ (200):**
```json
[
  {
    "id": 1,
    "title": "Flooding in downtown",
    "description": "Major flood warning in the city center",
    "center": {"lat": 55.7558, "lon": 37.6173},
    "radius": 500,
    "active": true,
    "created_at": "2026-01-01T12:00:00Z",
    "updated_at": "2026-01-01T12:00:00Z"
  }
]
```

#### PATCH /api/v1/incidents/{id}
Обновление инцидента (частичное).
```bash
curl -X PATCH http://localhost:8080/api/v1/incidents/1 \
  -H "Content-Type: application/json" \
  -H "X-API-Key: secret" \
  -d '{
    "title": "Updated flooding alert",
    "radius": 750
  }'
```
**Ответ (200):**
```json
{
  "id": 1,
  "title": "Updated flooding alert",
  "description": "Major flood warning in the city center",
  "center": {"lat": 55.7558, "lon": 37.6173},
  "radius": 750,
  "active": true,
  "created_at": "2026-01-01T12:00:00Z",
  "updated_at": "2026-01-01T12:00:00Z"
}
```

#### DELETE /api/v1/incidents/{id}
Деактивация инцидента.
```bash
curl -X DELETE -H "X-API-Key: secret" http://localhost:8080/api/v1/incidents/1
```
**Ответ (204):** Нет тела.

#### GET /api/v1/incidents/stats
Статистика уникальных пользователей за последние N минут.
```bash
curl -H "X-API-Key: secret" http://localhost:8080/api/v1/incidents/stats
```
**Ответ (200):**
```json
{
  "user_count": 67
}
```

## Архитектура вебхуков
Для надежности и транзакционности отправки вебхуков был реализован паттерн transactional outbox (https://microservices.io/patterns/data/transactional-outbox.html)
![pattern_image](https://microservices.io/i/patterns/data/ReliablePublication.png)