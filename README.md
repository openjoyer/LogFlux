# LogFlux

Для локальной разработки нужны Go 1.27, [Task v3](https://taskfile.dev/docs/installation) и [golangci-lint v2](https://golangci-lint.run/docs/welcome/install/). Для генерации protobuf также нужен `protoc`.

```sh
task deps       # скачать зависимости всех Go-модулей
task check      # форматирование, линтер, тесты и сборка
task fmt        # отформатировать Go-код
task proto      # сгенерировать Go-код из proto
```

Отдельные проверки: `task fmt:check`, `task lint`, `task test`, `task build`. Конфигурация линтера находится в `.golangci.yml`; CI запускает те же задачи через Task.
