## Компонент для работы со Swagger

### Использование

Для добавления компонента сваггер необходимо сгенерировать спецификацию (см. ниже), и передать в `WithSwagger` с помощью `swagger.LoadOpenAPISpec("")`, (передается путь к файлам proto, по умолчанию путь api/grpc)

Пример:
````go

...
vaar spec []byte
spec, _ = swagger.LoadOpenAPISpec("")

app, err := application.New(
	ctx, 
	application.WithHTTP(), 
	application.WithSwagger(spec), 
	application.WithGrpcServer[proto.UserServiceServer](proto.RegisterUserServiceServer, usrSrv),)
...

mux := http.NewServeMux()
app.RegisterRouter(mux)
````

Для регистрации gRPC серверов в gRPC Gateway можно использовать `WithSwaggerGatewayServer`, передаем RegisterXxxServiceHandlerServer и server impl. Необходимо использовать после `WithSwagger`.

Пример:
````go
...
app, err := application.New(
	ctx, 
	application.WithHTTP(), 
	application.WithSwagger(spec), 
	application.WithSwaggerGatewayServer(proto.RegisterRoleServiceHandlerServer, roleSrv),
	application.WithSwaggerGatewayServer(proto.RegisterUserServiceHandlerServer, usrSrv), 
	application.WithGrpcServer[proto.UserServiceServer](proto.RegisterUserServiceServer, usrSrv), 
	application.WithGrpcServer[proto.RoleServiceServer](proto.RegisterRoleServiceServer, roleSrv),
...

mux := http.NewServeMux()
app.RegisterRouter(mux)
````

Для регистрации только публичных серверов, используем `WithPublicSwaggerGateway`. Работает аналогично, только в случае передачи приватного grpc сервера, он не регистрирует его.

Для регистрации  gRPC серверов в gRPC Gateway без `WithSwaggerGatewayServer`:
````go
...
mux := http.NewServeMux()
	gw := runtime.NewServeMux()
	if err := proto.RegisterRoleServiceHandlerServer(ctx, gw, roleSrv); err != nil {
		logger.Fatal(ctx, "register gateway failed", logger.Err(err))
		return
	}
	if err := proto.RegisterUserServiceHandlerServer(ctx, gw, usrSrv); err != nil {
		logger.Fatal(ctx, "register gateway failed", logger.Err(err))
		return
	}
	mux.Handle("/", gw)
	
	app.RegisterRouter(mux)
...
````

### Спецификация

Пример глобальных метаданных
````protobuf
syntax = "proto3";

package openapi.v1;

option go_package = "proto";

import "protoc-gen-openapiv2/options/annotations.proto";

option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  info: {
    title: "BaseApp API";
    version: "v1";
    description: "Описание API"
  };
  base_path: "/api";                // UI будет идти в /api/...
  consumes: "application/json";
  produces: "application/json";
  tags: [
    { name: "Users", description: "Операции с пользователями" },
    { name: "Roles", description: "Роли и права" }
  ];
};
````

Как задать HTTP-метод и путь:
- Использовать `option (google.api.http) = { get|post|put|patch|delete: "/path" }`
- Параметры пути в `{name}` и то же имя поля в Request-сообщении
- Несколько альтернативных маршрутов — через additional_bindings:
````protobuf
option (google.api.http) = {
patch: "/api/v1/users/{id}"
body: "patch"
};
````

Как разделить path/query/body:
- GET/DELETE: тела нет, все поля запроса, не попавшие в {} пути, становятся query
- POST/PUT/PATCH: body: "*" — весь Request уходит в body; body: "field_name", только указанное подсообщение в body, остальные поля запроса становятся query

Как описывать поля (description/example/format)

На любом поле сообщения:
````protobuf
string email = 2 [
(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
description: "Email пользователя",
example: "\"user@example.com\"",
format: "email"
}
];
````

Как задавать HTTP-коды/ответы

На RPC:
````protobuf
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
responses: { key: "201" value: { description: "Создан" } }
responses: { key: "400" value: { description: "Bad Request" } }
responses: { key: "404" value: { description: "Not Found" } }
};
````

## Пример сервиса — все методы (GET/POST/PATCH/DELETE)

````protobuf
syntax = "proto3";
package user.v2;

option go_package = "git.vepay.dev/your/module/pkg/gateway/grpc/user/v2;userpb";

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
import "google/protobuf/empty.proto";

// Схемы

message User {
  string id = 1 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Идентификатор пользователя", example: "\"u_123\""
  }];
  string email = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Email", example: "\"user@example.com\""
  }];
  string first_name = 3 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Имя", example: "\"Иван\""
  }];
  string last_name = 4 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Фамилия", example: "\"Иванов\""
  }];
  int32 age = 5 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Возраст", example: "30", format: "int32"
  }];
}

message GetUserRequest {
  string id = 1 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "ID из пути", example: "\"u_123\""
  }];
  // Все НЕ вошедшие в body поля для GET становятся query-параметрами:
  bool verbose = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Подробный ответ", example: "true"
  }];
}

message CreateUserRequest {
  User user = 1 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Данные создаваемого пользователя"
  }];
  bool send_welcome = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Отправить приветственное письмо (query)", example: "true"
  }];
}

message UpdateUserRequest {
  string id = 1 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "ID из пути", example: "\"u_123\""
  }];
  // PATCH body: либо частичная модель, либо update_mask (как решите)
  User patch = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Частичные поля для обновления"
  }];
}

message ListUsersRequest {
  int32 page = 1 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Номер страницы (query)", example: "1", format: "int32"
  }];
  int32 page_size = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Размер страницы (query)", example: "50", format: "int32"
  }];
  repeated string role = 3 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Фильтр по ролям (multi-query)", example: "[\"admin\",\"editor\"]"
  }];
}

message ListUsersResponse {
  repeated User items = 1;
  int32 total = 2 [(grpc.gateway.protoc_gen_openapiv2.options.openapiv2_field) = {
    description: "Всего записей", example: "137"
  }];
}

// Сервис

service UserService {

  // GET /api/v1/users/{id}?verbose=true
  rpc GetUser(GetUserRequest) returns (User) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Пользователь по ID"
      description: "Возвращает пользователя по идентификатору"
      tags: "Users"
      responses: {
        key: "200"
        value: { description: "OK" }
      }
      responses: {
        key: "404"
        value: { description: "Не найден" }
      }
    };
  }

  // GET /api/v1/users?page=1&page_size=50&role=admin&role=editor
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
    option (google.api.http) = {
      get: "/api/v1/users"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Список пользователей"
      tags: "Users"
      responses: { key: "200" value: { description: "OK" } }
    };
  }

  // POST /api/v1/users
  rpc CreateUser(CreateUserRequest) returns (User) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "user"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Создать пользователя"
      tags: "Users"
      responses: {
        key: "201"
        value: { description: "Создан" }
      }
      responses: { key: "400" value: { description: "Валидационная ошибка" } }
    };
  }

  // PATCH /api/v1/users/{id}   (body = patch)
  rpc UpdateUser(UpdateUserRequest) returns (User) {
    option (google.api.http) = {
      patch: "/api/v1/users/{id}"
      body: "patch"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Частичное обновление"
      tags: "Users"
      responses: { key: "200" value: { description: "Обновлен" } }
      responses: { key: "404" value: { description: "Не найден" } }
    };
  }

  // DELETE /api/v1/users/{id}
  rpc DeleteUser(GetUserRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/users/{id}"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
      summary: "Удалить пользователя"
      tags: "Users"
      responses: { key: "204" value: { description: "Удален, тело пустое" } }
      responses: { key: "404" value: { description: "Не найден" } }
    };
  }
}
````





