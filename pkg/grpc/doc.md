## GRPC

### Конфигурация

gRPC Server

`grpc.server.private.host           ("" — слушать на всех)
grpc.server.port                  (по умолчанию "50051")
grpc.server.max_recv_msg_size     (по умолчанию 4MiB)
grpc.server.max_send_msg_size     (по умолчанию 4MiB)
grpc.server.connection_timeout    (по умолчанию 120s)
grpc.server.keepalive_time        (по умолчанию 30s)
grpc.server.keepalive_timeout     (по умолчанию 10s)`

gRPC Client (grpc_client_config.go)

`grpc.client.{name}.address            (обязателен)
grpc.client.{name}.timeout            (по умолчанию 30s)
grpc.client.{name}.max_recv_msg_size  (по умолчанию 4MiB)
grpc.client.{name}.max_send_msg_size  (по умолчанию 4MiB)
grpc.client.{name}.keepalive_time     (по умолчанию 30s)
grpc.client.{name}.keepalive_timeout  (по умолчанию 10s)`

Где {name} — это serviceName, который указываем в WithGrpcClient. Также принимается формат "grpc.client.{name}"

### Использование

#### Сервер

Для регистрации публичного сервера используем `WithPublicGrpcServer[]()`, для регистрации приватного, используем `WithPrivateGrpcServer[]()`
````go
// main.go (сервер)
usrSrv := &userserver.UserServer{}

app, err := application.New(
ctx,
// ... другие компоненты (metrics, trace, s3 и т.д.)
application.WithPublicGrpcServer[userv2.UserServiceServer](userv2.RegisterUserServiceServer, usrSrv),
)
if err != nil { /* handle */ }

if err := app.Run(); err != nil { /* handle */ }
````

Добавить свои интерсепторы на сервер, после application.New(...), до app.Run():
````go
app.PrivateGrpcServer.AddUnaryInterceptor(myUnary)
app.PublicGrpcServer.AddStreamInterceptor(myStream)
````

Добавить интерсепторы во время application.New(...)
````go
app, _ := application.New(

application.WithPrivateGrpcServer[rolev2.RoleServiceServer](rolev2.RegisterRoleServiceServer, roleSrv),
application.WithPrivateGrpcUnaryInterceptor(myAuth),
application.WithPrivateGrpcStreamInterceptor(rateLimit),

application.WithPublicGrpcServer[userv2.UserServiceServer](userv2.RegisterUserServiceServer, usrSrv),
application.WithPublicGrpcUnaryInterceptor(myAuth),
)
````

#### Клиент

````go
// main.go (клиент)
app, err := application.New(
    ctx, 
	application.WithGrpcClient[userv2.RoleServiceClient](
		"grpc.client.user_service",
		userv2.NewUserServiceClient, 
		),
    // ... http, metrics, trace и т.д.
)
if err != nil { /* handle */ }
````

### Метрики

- На сервере: MetricsUnaryInterceptor/MetricsStreamInterceptor (в pkg/grpc/server/interceptors).
- На клиенте: аналогичные метрики (в pkg/grpc/client/interceptors).

Типовые метрики:

- grpc_server_requests_total{method,status}
- grpc_server_request_duration_seconds{method}
- grpc_client_requests_total{method,status}
- grpc_client_request_duration_seconds{method}

### Трейсинг

Сервер: grpc.StatsHandler(otelgrpc.NewServerHandler(...)).

Клиент: grpc.WithStatsHandler(otelgrpc.NewClientHandler(...)).

### Логирование

pkg/grpc/logger.go перекидывает grpclog в logger:
`grpc.EnableWithContext(ctx)` и на сервере, и на клиенте


Переменные окружения:

log.level — минимальная «важность» (INFO/WARN/ERROR/FATAL)

## Примеры

Сервер
```go
func main() {
	log.Println("Starting User gRPC Server...")
	ctx := context.Background()
	
	usrSrv := &userserver.UserServer{}
	roleSrv := &roleserver.RoleServer{}
	
	app, err := application.New(
		ctx, 
		application.WithMetrics(), 
		application.WithTrace(), 
		application.WithS3(), 
		application.WithGrpcServer2[userv2.UserServiceServer](userv2.RegisterUserServiceServer, usrSrv), 
		application.WithGrpcServer2[rolev2.RoleServiceServer](rolev2.RegisterRoleServiceServer, roleSrv), 
		)

    if err != nil {
        logger.Fatal(ctx, "failed create app", logger.Err(err))
        return
    }

    di.RegisterFactory(ctx, repository.NewUserRepository)
    di.RegisterFactory(ctx, rolerepo.NewRoleRepository)
    di.Register[domain.IUserService](ctx, &service.UserService{})
    di.Register[roledomain.IRoleService](ctx, &roleservice.RoleService{})
    di.Register[userv2.UserServiceServer](ctx, usrSrv)
    di.Register[rolev2.RoleServiceServer](ctx, roleSrv)
	

    if err := app.Run(); err != nil {
        logger.Fatal(ctx, "failed to run app", logger.Err(err))
    }
}

```

Клиент
```go
func main() {
    ctx := context.Background()
    
    app, err := application.New(
        ctx,
        application.WithMetrics(),
        application.WithTrace(),
        application.WithHTTP(),
        application.WithGrpcClient[userv2.UserServiceClient]("grpc.client.user_service", userv2.NewUserServiceClient),
        application.WithGrpcClient[rolev2.RoleServiceClient]("grpc.client.user_service", rolev2.NewRoleServiceClient),
    )
    
    if err != nil {
        logger.Fatal(ctx, "failed create app", logger.Err(err))
        return
    }
    
    di.Register[handler.IUserHandler](ctx, &handler.UserHandlerGrpc{})
    di.Register[rolehandler.IRoleHandler](ctx, &rolehandler.RoleHandlerGrpc{})
    userHandler := di.Resolve[handler.IUserHandler](ctx)
    roleHandler := di.Resolve[rolehandler.IRoleHandler](ctx)
    
    e := echo.New()
    e.GET("/users/:id", userHandler.GetProfile)
    e.PUT("/users/:id", userHandler.UpdateProfile)
    e.POST("/users", userHandler.CreateProfile)
    e.POST("/upload", userHandler.UploadDoc)
    
    e.POST("/roles/assign", roleHandler.Assign)
    e.GET("/users/:id/roles", roleHandler.ListByUser)
    
    app.RegisterRouter(e)
    
    logger.Info(ctx, "Starting User HTTP Gateway")
    
    if err := app.Run(); err != nil {
        logger.Fatal(ctx, "failed to run app", logger.Err(err))
    }
    
    }
```