"""Я собираю исходный ZIP только из явного списка, без данных и истории Git."""
import hashlib
from pathlib import Path
import re
import sys
import zipfile

for stream in (sys.stdout, sys.stderr):
    if hasattr(stream, 'reconfigure'):
        stream.reconfigure(encoding='utf-8')

ROOT = Path(__file__).resolve().parents[1]
names = (ROOT / 'PUBLIC_FILES.txt').read_text(encoding='utf-8').splitlines()
if names != sorted(set(names)):
    raise SystemExit('PUBLIC_FILES должен быть уникальным и отсортированным')
payload = {}
for name in names:
    path = ROOT / name
    if Path(name).is_absolute() or '..' in Path(name).parts or path.is_symlink() or not path.is_file():
        raise SystemExit('Небезопасный путь публикации')
    data = path.read_bytes()
    text = data.decode('utf-8')
    if re.search(r'/(?:home|Users)/|[A-Z]:\\Users\\|[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}|gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,}|-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----', text):
        raise SystemExit('Возможные приватные данные: ' + name)
    payload[name] = data
destination = ROOT / 'artifacts'
destination.mkdir(exist_ok=True)
archive = destination / 'go-csv-quality-0.1.0.zip'
with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as output:
    for name, data in payload.items():
        info = zipfile.ZipInfo('go-csv-quality-0.1.0/' + name, (2026,10,25,0,0,0))
        info.compress_type = zipfile.ZIP_DEFLATED
        info.external_attr = 0o100644 << 16
        output.writestr(info, data)
digest = hashlib.sha256(archive.read_bytes()).hexdigest()
(destination / 'SHA256SUMS').write_text(f'{digest}  {archive.name}\n', encoding='ascii')
with zipfile.ZipFile(archive) as packed:
    expected = {'go-csv-quality-0.1.0/' + name for name in names}
    assert set(packed.namelist()) == expected
    for name, data in payload.items():
        assert packed.read('go-csv-quality-0.1.0/' + name) == data
print(f'Исходников: {len(payload)}; SHA256: {digest}')
