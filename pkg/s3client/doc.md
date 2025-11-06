## Компонент для работы с S3 хранилищем

### Общее описание

Клиент реализован в виде пакета https://git.vepay.dev/bakend-platform/s3client, в качестве базового пакета использован https://github.com/minio/minio-go.

Представлен интерфейсом:

```go
type Client interface {
    // Загрузка файла
	Upload(ctx context.Context, input *UploadInput) error

    // Скачивание файла
	Download(ctx context.Context, key string) (*DownloadOutput, error)

    // Проверка существования файла
	Exist(ctx context.Context, key string) (bool, error)

    // Удаление файла
	Delete(ctx context.Context, key string) error

    // Перемещение файла
	Move(ctx context.Context, sourceKey, destinationKey string) error

    // Получение временной ссылки на скачивание файла
	PresignedGetObject(ctx context.Context, key string, ttl time.Duration) (string, error)

    // Проверка коннекта к S3 хранилищу
	Ping(ctx context.Context) error

    // Выгрузка из памяти хелс-чеков
	Close() error
}
```


Для создания клиента используется конструктор `NewClient` который принимает конфиг:

```go
type Config struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	Region          string `json:"region" yaml:"region"`
	UseSSL          bool   `json:"use_ssl" yaml:"use_ssl"`
	CreateBucket    bool   `json:"create_bucket" yaml:"create_bucket"`
}
```

При использовании в комбинации с base-app, использует параметры из env-а:

- s3.endpoint хост хранилища 
- s3.access_key_id и s3.secret_access_key - креды доступа
- s3.bucket название бакета (если не указан то берется из APP_NAME)
- s3.region регион хранения, по умолчанию us-east-1
- s3.use_ssl флаг использования защищенного соединения
- s3.create_bucket если true то бакет с именем S3_BUCKET будет создан автоматически, если он не создан

### Примеры использования

#### Использование в связке с base-app

```go

    ctx := context.Background()

	app, err := application.New(
		ctx,
		application.WithHTTP(),
		application.WithS3(),
	)

	//...

    func someUseCase(ctx context.Context, key string) {
        err := app.S3.Upload(ctx, &s3client.UploadInput{
			Key:         "demo/example.txt",
			Body:        strings.NewReader("Hello from S3 Client Demo!\nThis is a test file content."),
			ContentType: "text/plain",
			Metadata: map[string]string{
				"created-by": "demo-app",
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		})
    }

```