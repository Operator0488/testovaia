## Компонент Wokflow 

### Общее описание

Компонент позволяет реализовать распределенный во времени или между сервисами бизнес-процесс. В основе подхода с бизнес-процессом лежит снятие ответстенности за принятие решения с сервиса или логического блока внутри сервиса. 

### Принцип работы

Бизнес-процесс формируется в виде графической схемы с BPMN-нотацией. Схема представляет из себя граф из последовательно вызываемых задач. Задачи - это короткие легковесные функции, реализуемые на стороне сервисов. Такие функции не принимают решения, а лишь выполняют небольшую работу в рамках бизнес-процесса. К сервисам задачи поставляются через специальный консумер TaskHandler. Задачи забираются через pull модель. Кроме делегирования выполненя задач, бизнес-процесс может получать информацию и извне. Это достигается через события. Примером может быть действие пользователя в рамках бизнес-процесса. Событие о этом действии позволяет бизнес-процессу отреагировать и продолжить выполнение в нужном направлении.  

### Данные процесса

Каждый бизнес-процесс имеет свой скоуп данных. Скоуп представляет из себя набор вида ключ-значение. Количество данных в скоупе не ограничено, но должно подчиняться здравому смыслу. Скоуп не должен содержать избыточной информации и тем более не должен содержать данные, которые могут стать неконсистентными. Хороший пример для данных скоупа: идентификатор польззователя, идентификатор заказа, флаги управления. Плохой пример данных скоупа: номер телефона пользователя, баланс пользователя.

В конечном счете скоуп процесса должен содержать минимально достаточное количество данных, чтобы на их основании получить всю остальную консистентную информацию непосредственно на стороне сервиса при выполнении задачи. Такой подход к скоупу не создает параллельной информации и является единственно верным. 

Скоуп процесса формируется при его запуске. Переменные скоупа могут быть добавлены или изменены при завершении задачи или при генерации события.  

### Допустимые элементы в схеме

##### ServiceTask
Основной элемент бизнес-процесса. Каждый элемент serviceTask должен иметь уникальный текстовый идентификатор. Этот текстовый идентификатор позволяет сервисам подписываться на задачи по типу. Аналогией может служить имя очереди в RMQ или имя топка Kafka. Хороший прием - формировать имя serviceTask по шаблоу "processName_taskName"  

##### Gateway
Логическая развилка. Может быть развилкой по условию, элементом распараллеливания логики, либо точкой выбора дальнейшей логики по триггеру (таймер или событие) 

##### Timer 
Автономный элемент для формирования события в бизнес-процессе. Время срабатывания таймера может быть как константным, так и быть переданным через связанную переменную скоупа процесса.

##### Event
Позволяет сигнализировать бизнес-процессу о внешнем событии. Ожидание события переводит бизнес-процесс в Idle режим. События, аналогично TaskHandler, должны иметь уникальное имя. Хорошей практикой для наименования является паттерн "processName_eventName"

### Начало работы

1. Установить Camunda Modeler или использовать его partial версию. Доступно по ссылке https://camunda.com/download/modeler/
2. Для локальной разработки и проверки схемы развернуть в docker минимальный рабочий стенд Zeebe. Можно использовать compose ниже
```dockerfile
services:

  zeebe: # https://docs.camunda.io/docs/self-managed/platform-deployment/docker/#zeebe
    image: camunda/zeebe:8.4.1
    container_name: zeebe
    ports:
      - "26500:26500"
      - "9600:9600"
    environment: # https://docs.camunda.io/docs/self-managed/zeebe-deployment/configuration/environment-variables/
      - ZEEBE_BROKER_EXPORTERS_ELASTICSEARCH_CLASSNAME=io.camunda.zeebe.exporter.ElasticsearchExporter
      - ZEEBE_BROKER_EXPORTERS_ELASTICSEARCH_ARGS_URL=http://elasticsearch:9200
      # default is 1000, see here: https://github.com/camunda/zeebe/blob/main/exporters/elasticsearch-exporter/src/main/java/io/camunda/zeebe/exporter/ElasticsearchExporterConfiguration.java#L259
      - ZEEBE_BROKER_EXPORTERS_ELASTICSEARCH_ARGS_BULK_SIZE=1
      # allow running with low disk space
      - ZEEBE_BROKER_DATA_DISKUSAGECOMMANDWATERMARK=0.998
      - ZEEBE_BROKER_DATA_DISKUSAGEREPLICATIONWATERMARK=0.999
      - "JAVA_TOOL_OPTIONS=-Xms512m -Xmx512m"
    restart: always
    healthcheck:
      test: [ "CMD-SHELL", "timeout 10s bash -c ':> /dev/tcp/127.0.0.1/9600' || exit 1" ]
      interval: 30s
      timeout: 5s
      retries: 5
      start_period: 30s
    volumes:
      - zeebe:/usr/local/zeebe/data
    depends_on:
      - elasticsearch

  operate: # https://docs.camunda.io/docs/self-managed/platform-deployment/docker/#operate
    image: camunda/operate:8.4.1
    container_name: operate
    ports:
      - "8081:8080"
    environment: # https://docs.camunda.io/docs/self-managed/operate-deployment/configuration/
      - CAMUNDA_OPERATE_ZEEBE_GATEWAYADDRESS=zeebe:26500
      - CAMUNDA_OPERATE_ELASTICSEARCH_URL=http://elasticsearch:9200
      - CAMUNDA_OPERATE_ZEEBEELASTICSEARCH_URL=http://elasticsearch:9200
      - management.endpoints.web.exposure.include=health
      - management.endpoint.health.probes.enabled=true
    healthcheck:
      test: [ "CMD-SHELL", "wget -O - -q 'http://localhost:8080/actuator/health/readiness'" ]
      interval: 30s
      timeout: 1s
      retries: 5
      start_period: 30s
    depends_on:
      - zeebe
      - elasticsearch

  elasticsearch: 
    image: docker.elastic.co/elasticsearch/elasticsearch:8.12.0
    container_name: elasticsearch
    ports:
      - "9200:9200"
      - "9300:9300"
    environment:
      - bootstrap.memory_lock=true
      - discovery.type=single-node
      - xpack.security.enabled=false
      # allow running with low disk space
      - cluster.routing.allocation.disk.threshold_enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ulimits:
      memlock:
        soft: -1
        hard: -1
    restart: always
    healthcheck:
      test: [ "CMD-SHELL", "curl -f http://localhost:9200/_cat/health | grep -q green" ]
      interval: 30s
      timeout: 5s
      retries: 3
    volumes:
      - elastic:/usr/share/elasticsearch/data

volumes:
  zeebe:
  elastic:
```

### Инициализация компонента

Инициализация и запуск компонента происходит в два шага. Первый шаг - создание специального билдера для предварительной настройки, декларации процессов и подписок на задачи. Второй шаг - запуск подготовленного компонента.

```go
// Билдер для предварительной настройки компонента
workflowBuilder := NewWorkflowBuilder(
    "myServiceName", 
    "https://zeebe",
    "demo",
    "demo")

// Декларация процесса (экспортирует или обновляет схему процесса)
workflowBuidler.WithProcess("bmpn/myServiceProcess.bpmn")

// Запускает компонент и возвращает интерфейс управления процессами
// Метод не является блокирующим в случае недоступности сервера Workflow
workflow := workflowBuilder.Run(context.Background())
```

### Подска на задачи

Задача реализуется через функцию  
```go
type WorkflowTaskHandler func(ctx context.Context, task WorkflowTask) error
```
Подписка осуществляется через WorkflowBuilder методом WithHandler
```go
type MyTaskHandler struct {
    notifyService NotifyService
}

func (hdlr *MyTashHandler) ResolveDeps(notifyService NotifyService) {
    hdlr.notifyService = notifyService
}

func (hdlr *MyTashHandler) Handle(ctx context.Context, task workflow.WorkflowTask) error {
    userId, ok := task.GetScopeVariable("userId")
	if !ok {
	    return workflow.MakeIncident("missed userId in process scope")	
    }
	
	return hdlr.notifyService.SendCompleteRegistrationEmail(ctx, userId)
}

// Обработчик задачи
myTaskHandler := di.Register(ctx, &MyTashHandler{})

// Регистрация обработчика для задачи по типу
workflowBuidler.WithHandler(
    myServiceProcess_TaskName, 
    myTaskHandler.Handle,
    // Опционально. Можно не передавать этот аргумент.
    // В данном случае изменение количества ретраев при ошибке до постановки на инцидент
    &workflow.TaskHandlerConfig{
        IncidentMaxRetries: 100	
    })
```

В случае возврата ошибки задача вернется в работу через заданный таймаут. Стандартный таймаут может быть изменен при подписке на задачу. 

Если задача определенное количество раз возвращается на retry, процесс может быть переведен в режим инцидента. В этом состоянии процесс ожидает ручного вмешательства для устранения причин невозможности дальнейшего выполнения. Количество ретраев перед переводом в состояние инцидента может быть изменено при подписке на задачу.

Процесс также может быть переведен в режим инцидента принудительно через вовзрат ошибки workflow.ErrWorkflowIncident. Это может быть полезно, если при выполнении задачи зафиксировано неопределенное поведение, неконсистентность данных или невозможность выполнения задачи. Любая иная причина, при которой возобновление попыток выполнить задачу становится бессмесленным. Процесс переводится в состояние инцидента, алертит в соответствующие каналы и ожидает вмешательство дежурного.