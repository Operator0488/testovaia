## Пакет db

Настройки подключения к Postgres описаны в конфигурационном файле [в документе](../../docs/config.md)

### Миграции c atlas

При старте приложения миграции автоматически накатываются (если есть), управляется настройкой `postgres.migrate`, по дефолту включен.

Для работы с миграциями испольуется (https://atlasgo.io/getting-started#installation)[atlas]
Его необходимо установить, так же должен быть установлен `docker`

Инструкция по командам https://atlasgo.io/cli-reference

Что нужно сделать чтобы миграции заработали локально:
1) Установить `atlas`
2) Настроить конфиг файл `atlas.hcl` (есть в репозитории темплейта https://git.vepay.dev/knoknok/backend-service-template)

Пример такого конфига:

```sh

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./internal/domain/entity", // папка в которой лежат gorm-модели 
    "--dialect", "postgres",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  migration {
    dir = "file://./migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

```