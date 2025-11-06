## Описание конфига приложения

Полный конфиг приложения со всеми компонентами которые могут быть включены в base-app.
Большинство настроек может быть переопределено через config-server (consul) и vault при СТАРТЕ приложения.
Если необходимо перезагружать конфиги компонентов в рантайме, то нужно использовать доработать компонент `IConfigWatcher` в base-app.
По умолчанию названия разделов в vault и consul для приложения берутся из `app.name`
Метки: 

1) `const` - обязательный параметр, зашивается в `config.yaml`
2) `required` - обязательный параметр (может быть переопределен через config-server) если компонент включен в base-app
3) `vault-shared` - параметр нужно хранить в vault в shared разделе
4) `consul-shared` - параметр хранится в shared разделе 
5) `vault` - параметр нужно хранить в vault в разделе самого приложения

```yaml
# Базовые настройки приложения
app:
  name: "super-app"     # [const, required], название приложения, данная настройка будет использоваться в качестве дефолта для: (секция в vault, секция в consul, бакет в S3)
  port: 8080            # порт на котором будет подниматься http-сервер

# Настройки доступа к S3
s3: 
  endpoint: "localhost:9000"        # [required, consul-shared], адрес s3, эта настройка может быть переопределена через consul или vault
  bucket: "super-app"               # бакет в который будут складываться файлы, по дефолту app.name
  region: "us-east-1"               # регион, по дефолту us-east-1
  use_ssl: false                    # ssl
  create_bucket: true               # автоматическое создание бакета если бакет не найден
  access_key_id: "minioadmin"       # [required, vault-shared] логин доступа
  secret_access_key: "minioadmin"   # [required, vault-shared] пароль доступа

# Настройки доступа к Kafka
kafka:
  brokers: "kafka:9092"             # [required, consul-shared] брокеры через запятую
  group: "super-app"                # косумер группа, по дефолту app.name

# Настройки Redis
redis:
  addrs: "localhost:6379"           # [required, consul-shared] адрес сервера
  db: ""                            # [required, consul-shared] адрес бд
  pool_size: ""                     # ?
  dial_timeout: ""                  # ?
  read_timeout: ""                  # ?
  write_timeout: ""                 # ?
  username: ""                      # [required, vault-shared] логин доступа
  pwd  : ""                         # [required, vault-shared] пароль доступа

# Настройки prometheus
metrics:
  addr: "localhost"                 # [required, consul-shared] адрес
  port: "9090"                      # порт, по дефолту 9090

# Настройка трейсинга
trace:
  endpoint: "localhost"             # [required, consul-shared] адрес
  insecure: false                   # защищенность соединения
  sample_ratio: 0.5                 # sample_ratio
  sevice_name: "super-app"          # название сервиса, по дефолту app.name 
  env: ""                           # ?
  service_ver: ""                   # версия приложения
  protocol: "grpc"                  # протокол otlp либо grpc либо http

# Настройки БД postgres
postgres:
  dsn: "addr"                       # [required, vault], готовый адрес подключения к бд

  # если не указан dsn то он будет собираться из отдельных параметров
  host: "postgres:5432"             # [required, consul], адрес бд
  port: 5432                        # [required, consul], порт
  user: "app_user"                  # [required, vault], пользователь
  password: "app_password"          # [required, vault], пароль
  database: "app_db"                # [required, consul], название бд
  sslmode: "disable"                # [consul], режим соединения enable, disable

  # другие настройки
  max_open_conns: 25                # настройки пула соединений
  max_idle_conns: 25                # настройки пула соединений
  conn_max_lifetime: "5m"           # настройки пула соединений
  slow_threshold: "200ms"           # настройки трейсинга для долгих запросов
  health_check_interval: "30s"      # интервал проверки соединения
  migrate: true                     # запуск миграции при старте приложения

# Настройки конфиг-сервера
consul:
  app_prefix: "super-app"           # префикс (папка) в котором лежат настройки приложения (по дефолту app.name), поддерживаются только простые поля (строки, числа), поддержка json,yaml,hcl не предусмотрена
  shared_prefix: "shared"           # префикс (папка) в котором лежат общие настройки для всех приложений, например настройки к Kafka, S3 и тд (по дефолту shared)

# Настройки vault
vault:
  app_path: "super-app"             # папка с секретами приложения (по дефолту app.name)
  shared_path: "shared"             # папка с общими секретами всех приложений (по дефолту shared)
  
# Провайдер для хранения переводов
tolgee:
  enabled: true                     # включена возможность или нет
  host: "http://tolgee:8089"        # хост с которого доступен tolgee
  project_id: 1                     # [required, consul] идентификатор проекта в котором лежат переводы
  tags: ["backend"]                 # теги которые будут добавляться к переводам
  api_key: "xxxx"                   # [required, vault] api-ключ для доступа к tolgee

```

### Другие параметры которые могут быть переданы через env

1) `VAULT_ADDR` - адрес vault, по дефолту `vault:8200`
2) `VAULT_TOKEN_PATH` - токен для доступа к VAULT
3) `VAULT_MOUNT_PATH` - точка монтирования для Key Value хранилища, по дефолту `sercrets`
4) `VAULT_DISABLED` - если "true" то при запуске приложения подключение к vault будет скипаться
5) `CONSUL_ADDR` - адрес конфиг-сервера, по дефолту `consul:8500`
6) `CONSUL_TOKEN` - токен доступа к консулу
7) `CONSUL_TOKEN_PATH` - файл с токеном доступа
8) `CONSUL_DISABLED` - если "true" то при запуске приложения подключение к consul будет скипаться
9) `LOG_LEVEL` - какие логи отображаем, может принимать значения: `debug`,`info`,`warn`,`error`