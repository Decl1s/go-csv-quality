# Технологии, которые я использую

| Компонент | Закреплённая/фактическая версия | Назначение и мой выбор | Официальный источник |
| --- | --- | --- | --- |
| Go | 1.26.8, go.mod | Небольшой CLI binary и потоковая обработка без runtime зависимостей | [Release notes](https://go.dev/doc/go1.26) |
| encoding/csv | stdlib Go 1.26.8 | Корректный quoting/CRLF вместо split по запятой | [csv](https://pkg.go.dev/encoding/csv) |
| io.LimitedReader | stdlib Go 1.26.8 | Ограничение фактически прочитанного потока | [io](https://pkg.go.dev/io#LimitedReader) |
| crypto/sha256, encoding/binary/hex/json | stdlib Go 1.26.8 | File/row identity, length framing и отчёты | [sha256](https://pkg.go.dev/crypto/sha256), [binary](https://pkg.go.dev/encoding/binary), [json](https://pkg.go.dev/encoding/json) |
| flag, os, filepath, fmt | stdlib Go 1.26.8 | CLI, lock/stage/rename и стабильные exit codes | [os](https://pkg.go.dev/os), [flag](https://pkg.go.dev/flag) |
| regexp, strconv, time, utf8, math, errors | stdlib Go 1.26.8 | Строгий контракт полей, даты, overflow и безопасные ошибки | [time](https://pkg.go.dev/time), [strconv](https://pkg.go.dev/strconv) |
| testing, bytes, strings | stdlib Go 1.26.8 | Контрактные, размерные и файловые проверки | [testing](https://pkg.go.dev/testing) |
| Python | локально 3.14.4, .python-version и CI pin | Только проверка собранного CLI и упаковка, приложению не нужен | [Python 3.14](https://docs.python.org/3.14/) |
| Python json/os/pathlib/subprocess/tempfile/hashlib/re/zipfile | stdlib Python 3.14.4 | Smoke, allowlist/privacy, воспроизводимый ZIP/SHA | [subprocess](https://docs.python.org/3.14/library/subprocess.html), [zipfile](https://docs.python.org/3.14/library/zipfile.html) |
| Git, GitHub Actions | Git 2.53.0; actions по SHA ниже | История одного проекта и матрица ОС | [Git](https://git-scm.com/docs), [Actions](https://docs.github.com/en/actions) |

У меня нет внешних Go modules или pip packages: прямых и транзитивных внешних
зависимостей ноль. Go standard library входит в toolchain; go.sum без сторонних
модулей не нужен. `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off` запрещают
неявную загрузку compiler/modules в проверках. Транзитивную stdlib я отдельно
как внешнюю библиотеку не выдаю. Локальный Git: фактическую версию фиксирую в VALIDATION.

Я закрепил action checkout `11d5960a326750d5838078e36cf38b85af677262`,
setup-go `924ae3a1cded613372ab5595356fb5720e22ba16`,
setup-python `a26af69be951a213d495a4c3e4e4022e16d87065`,
upload-artifact `ea165f8d65b6e75b540449e92b4886f43607fa02`.
Runner ubuntu-latest/windows-latest плавающий; реальное исполнение CI отмечаю
отдельно. Контейнеры, БД, очереди и платные сервисы не нужны этой задаче.
