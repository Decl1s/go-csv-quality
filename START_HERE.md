# Где я разместил код

| Файл | Что я в нём делаю | С чего начинаю чтение |
| --- | --- | --- |
| quality.go | Контракт строки, поток CSV, rejects, суммы и gate | validate → Process |
| main.go | Аргументы, синтетика и staging/lock/rename | command → publish |
| quality_test.go | Контракт, границы, fail gate, файловая публикация | TestDemoContract и TestFatalPublishLeavesNoResult |
| scripts/smoke.py | Собираю binary и прохожу exit codes/outputs | Весь короткий сценарий |
| scripts/package_release.py | Allowlist/privacy, ZIP и SHA256SUMS | PUBLIC_FILES → payload → archive |
| .github/workflows/ci.yml | Те же проверки Linux/Windows | Матрица, pins, env |

Я читаю бизнес-контракт CONCEPT до кода, затем NOTES до самостоятельной правки.
CLI не содержит правил строк; core не открывает файлы. `maxBadBP` — бюджет
качества в целых basis points, `seen` — ID только принятых записей, `bounded` —
ограничитель реально прочитанного потока, `stage` — ещё не опубликованный output.

Для новой валюты я меняю validate и стабильный список totals, контракт и demo/test.
Для нового правила начинаю с точного code/допустимого примера; не печатаю payload
в ошибке. Для составного ключа заменяю seen/order_id с явным grain, а не склейкой
строк без разделителей. Увеличение лимита файла не снимает ограничения памяти map.

Я предлагаю пять задач с ожидаемыми ответами в NOTES. Для эксперимента создаю
новую копию синтетического input и новый output. Порог не «исправляет» плохую
строку: он меняет gate, а accepted/rejected набор остаётся тем же.

VS Code workspace содержит явные задачи; F5/GUI отладку я не проверял.
