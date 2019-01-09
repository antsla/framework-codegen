Мини-фреймворк для кодогенерации http-обертки модели с валидацией.

Для генерации обертки выполните:

```
cd app && \
go build codegen.go && \
./codegen {filename}.go {filename_wrapper}.go
```

Для запуска тестового примера выполните:

`docker-compose up codegen`

Результирующий файл: `api_handlers.go`