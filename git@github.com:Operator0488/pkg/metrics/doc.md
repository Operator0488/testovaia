# Метрики

Код обновляет метрики → /metrics отдаёт их в Prometheus-формате → Prometheus периодически “скрейпит” → Grafana строит графики/алерты.

#### Где обновляются метрики:
- системные - стандартные коллекторы;
- Kafka - через middleware продюсера/консьюмера + фоновый lag-collector;
- DB - в обертках над запросами;
- Vault - при попытках чтения секретов (счетчики/ошибки);
- Zeebe - жду реализацию.

#### Ключевые моменты:

- Метрики инициализируются при старте BaseApp, компонент WithMetrics();
- GET /metrics - Prometheus endpoint;
- Векторные метрики (CounterVec/GaugeVec/HistogramVec) появятся после первой серии.

## Справочник метрик

#### Системные (из коробки):
- go_goroutines - количество goroutines
- go_threads - количество потоков
- go_gc_duration_seconds - статистика GC
- go_info - информация о версии Go
- go_memstats_alloc_bytes - аллокации памяти
- go_memstats_heap_* - различные метрики heap
- process_* - метрики процесса (CPU, память, файловые дескрипторы)

Больше метрик можно найти в [документации Prometheus по Go клиенту](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#hdr-Standard_Collectors).

#### Kafka:
- **kafka_messages_total{topic, type}** — counter type: produce|consume. Инкремент при успешной отправке/обработке
- **kafka_errors_total{topic, type}** — counter Ошибки продюсера/консьюмера (ретраи считаем отдельными ошибками)
- **kafka_latency_seconds_bucket{topic, type, le},** …_sum, …_count — histogram Время операции: отправка (producer) или обработка сообщения (consumer)
- **kafka_consumer_lag{topic}** — gauge Лаг по топикам: max(0, end_offset - committed_offset), обновляется фоново раз в 5 секунд` collectMetricsInterval   = 5 * time.Second` (взято из головы, можно и реже)

#### Redis:
- **redis_query_duration_seconds{command}** — HistogramVec по командам GET, SET, DEL (publish/subscribe не трекаем)
- **redis_connections{state}** — GaugeVec по состояниям пула open (открытые соединения), idle (не используется), in_use (обрабатывается), каждые 5 секунд смотрим в фоне const `collectMetricsInterval   = 5 * time.Second` (взято из головы, можно и реже)

#### Vault:
- **vault_secret_access_total{type, mount, path}** — counter Кол-во попыток чтения секрета, тип: kv pki. Обновляется в LoadKV/LoadPKI (обёртка withMetrics).
- **vault_errors_total{type, mount, path}** — counter Ошибки доступа сетевые, пустой ответ.. источник: та же обертка, инкремент при err != nil
- **vault_request_duration_seconds{type, mount, path, le}** — histogram Длительность запросов к Vault

## Grafana: ключевые панели (PromQL)

#### Kafka:
- Пропускная способность Kafka (топики): `sum by (topic) (rate(kafka_messages_total[1m]))`
- Error rate Kafka, %:` 100 * sum by (topic, type) (rate(kafka_errors_total[5m])) / (sum by (topic, type) (rate(kafka_messages_total[5m])) + 1e-9)`
- Latency p95 Kafka: `histogram_quantile(0.95,  sum by (topic, type, le) (rate(kafka_latency_seconds_bucket[5m])))`
- Consumer lag по топикам: `sum by (topic) (kafka_consumer_lag)`

#### Redis:
- Latency p95 по Redis: `histogram_quantile(0.95,  sum by (command, le) (redis_query_duration_seconds_bucket[5m])))`
- Пул соединений: `redis_connections{state="open"}`, `redis_connections{state="in_use"}`, `redis_connections{state="idle"}`
- Error rate Redis, %: `100 * sum(rate(redis_errors_total[5m])) / (sum(rate(redis_query_duration_seconds_count[5m])) + 1e-9)`

#### Vault:
- Error rate %: `100 * sum(rate(vault_errors_total[5m])) / sum(rate(vault_secret_access_total[5m]))`
- Error rate % mount: `100 * sum by (type, mount) (rate(vault_errors_total[5m])) / sum by (type, mount) (rate(vault_secret_access_total[5m]))`
- p95 latency (KV) по mount/path: `histogram_quantile(0.95,  sum by (mount, path, le (rate(vault_request_duration_seconds_bucket{type="kv"}[5m])))`


