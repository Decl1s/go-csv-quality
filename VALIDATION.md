# Что я фактически проверил

25.10.2026, локальный Linux amd64: Go 1.26.8, Python 3.14.4.
Я выполнил 17 Go test functions (включая 10 табличных subtests), go vet,
сборку binary и smoke реального CLI с exit 0/2/3. В demo подтверждены
8 records, 4 accepted, 4 rejected, EUR 3000/RUB 39000/USD 2000.
Проверил exact header/UTF8/date/числа/дубли, пустые данные, лимит файла/строк,
overflow, безопасный write failure, отсутствие partial output и no-overwrite.

Первая CLI сборка обнаружила inherited VCS boundary: Go пытался прочитать
чужой родительский Git и отказал при VCS stamping. Я создал собственный Git
и явно выключил stamping в portable build через -buildvcs=false.
После изменения smoke прошёл. Публичную историю и metadata в binary не встраиваю.

Публикация, удалённый CI, релиз и запуск из ZIP отражаются ниже по факту.
Я не называю локальные проверки исполнением Windows/macOS. Падение питания,
враждебные symlink races и forced crash между rename/fsync не проверял.

Я также проверил отказ при существующем lock и различие SHA для разных плохих
UTF-8 байтов. Go test повторён после изменения row framing, go vet прошёл.
Git 2.53.0; go list -m all содержит только текущий модуль, внешних зависимостей нет.

Я собрал ZIP из 19 allowlisted файлов, распаковал в новый временный каталог
и выполнил из него go test -count=1 и smoke собранного CLI: оба прошли.
Состав и байты каждого ZIP member совпали с исходниками.

## Первый удалённый CI и исправление Windows

[CI 40223653067](https://github.com/Decl1s/go-csv-quality/actions/runs/40223653067):
Ubuntu прошёл полностью. В Windows прошли Go tests/vet и binary smoke, но
упаковщик упал на русском print: stdout использовал cp1252 (UnicodeEncodeError).
Я явно переключил stdout/stderr упаковщика на UTF-8 и задал UTF-8 при чтении
PUBLIC_FILES. Локально повторил упаковку с PYTHONIOENCODING=cp1252; исправленный
скрипт отработал. Полный финальный результат матрицы проверяю перед релизом,
текущий статус виден в [Actions](https://github.com/Decl1s/go-csv-quality/actions).

## Подтверждённый результат после исправления

[CI 40223981376](https://github.com/Decl1s/go-csv-quality/actions/runs/40223981376)
завершился успешно в обеих задачах: ubuntu-latest и windows-latest. Я проверил
состояния всех шагов, включая tests, vet, binary smoke, упаковку и upload artifact.
Проверенный source commit: f19e54300d68ebb079efe5b092430b6ebd4eabca.
После него я добавил только эту запись документации и пересобрал ZIP по тому же
allowlist; Go и Python исходники не менял. Повторять программные проверки только
из-за записи результата не стал. Исходники публичные, 19 blob SHA сверены с GitHub.
Релиз и вложения дополнительно сверяю при публикации; его ссылка — раздел Releases.
