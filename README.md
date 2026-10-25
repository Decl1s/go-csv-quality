# Контроль качества CSV перед загрузкой

Я написал учебный quality gate для выгрузки заказов: принимаю CSV, отделяю
корректные записи, сохраняю безопасный список отказов и решаю, допустима ли доля
брака для дальнейшего конвейера. Новый основной инструмент — Go 1.26.8 и его
стандартная библиотека; внешних пакетов и фоновых служб нет.

Версия 0.1.0, 25.10.2026. Я использую только воспроизводимую синтетику.
Это учебный проект, а не коммерческая система или подтверждённый опыт клиента.

## Моя демонстрация на 5 минут

Нужен Go 1.26.8. В Linux/macOS из корня checkout:

```sh
mkdir -p workspace
GOTOOLCHAIN=local go build -buildvcs=false -o workspace/csv-quality .
./workspace/csv-quality demo -file workspace/demo.csv
./workspace/csv-quality check -input workspace/demo.csv -out workspace/strict
```

Последняя команда **возвращает exit 3 намеренно**: из 8 строк я принимаю 4,
отклоняю 4, а порог по умолчанию равен нулю. Это опубликованный отчёт с fail gate,
а не технический сбой. В `workspace/strict/` находятся `accepted.csv`,
`rejections.jsonl`, `report.json`. Для второго запуска выбираю другой output:

```sh
./workspace/csv-quality check -input workspace/demo.csv -out workspace/tolerant -max-bad-bp 5000
```

5000 basis points означают 50%; ровно 4/8 допускается, gate=pass, exit 0.
Итоги корректных заказов: RUB 39000, USD 2000, EUR 3000 minor units.
Я не складываю разные валюты в один total. В PowerShell:

```powershell
New-Item -ItemType Directory -Force workspace
go build -buildvcs=false -o workspace/csv-quality.exe .
./workspace/csv-quality.exe demo -file workspace/demo.csv
./workspace/csv-quality.exe check -input workspace/demo.csv -out workspace/strict
$LASTEXITCODE
```

## Что задаёт каждый аргумент

| Команда/параметр | Как я его трактую |
| --- | --- |
| `demo -file PATH` | Создаю новый синтетический CSV; существующий файл не заменяю |
| `check -input PATH` | Читаю только явно выбранный обычный файл |
| `-out DIR` | Публикую три результата в новом каталоге; parent должен существовать |
| `-max-bad-bp N` | Допустимая доля отказов 0…10000; сравниваю целыми без округления |
| `version` | Печатаю версию 0.1.0 |

Exit 0 — gate pass; exit 3 — gate fail с сохранённым отчётом; exit 2 — ошибка
параметров, структуры, I/O или публикации. Я не публикую частичный результат
при сломанном CSV quoting, превышении лимита или ошибке записи.
Повтор того же input в новом output воспроизводит байты; существующий output
сохраняется. Пустой набор после заголовка даёт fail даже при пороге 10000.

## Как я проверяю проект

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go test -v ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 go vet ./...
python3 scripts/smoke.py
python3 scripts/package_release.py
```

Для Python-скриптов я закрепил 3.14.4; в Windows использую `python` вместо
`python3`. Приложению Python не требуется: это помощник проверки/упаковки.
Я проверил локальный Linux, 17 Go test functions с 10 табличными subtests,
vet и собранную CLI. Удалённый CI и публикацию отражаю отдельно в VALIDATION;
наличие workflow не называю пройденной проверкой Windows.

## Как я читаю и изменяю код

[START_HERE](START_HERE.md) — карта, [CONCEPT](CONCEPT.md) — контракт,
[NOTES](NOTES.md) — подробный конспект с упражнениями,
[TECHNOLOGIES](TECHNOLOGIES.md) — версии и официальные источники,
[VALIDATION](VALIDATION.md) — фактические проверки.
Я могу открыть `go-csv-quality.code-workspace`; в нём только этот репозиторий,
четыре явные задачи редактора, без автоматического запуска и установки расширений.

Исходники: [Decl1s/go-csv-quality](https://github.com/Decl1s/go-csv-quality).
Архив и SHA256SUMS собираю в `artifacts/` по PUBLIC_FILES. Входы, outputs,
исполняемые файлы и Git history в публичный архив не включаю.

## Мои ограничения

Я ограничил вход 8 MiB и 100000 логических CSV-записей. ID запоминаю в памяти
до этого предела; целиком файл для обработки не загружаю. CSV parser может
материализовать большую запись до предела файла, поэтому память не O(1).
Глобальную бизнес-полноту, реальные персональные данные и свежесть источника
этот контракт не распознаёт. SHA не является анонимизацией.
Файловый lock защищает согласованно работающие экземпляры приложения, не враждебное
изменение parent-каталога. При падении процесса могут остаться lock/stage;
автоматически чужие файлы не удаляю. Power-loss durability каталога не обещаю.
