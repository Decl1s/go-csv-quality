"""Я проверяю собранную CLI-программу и её файловые границы на синтетике."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='csv-quality-smoke-') as folder:
    work = Path(folder)
    env = dict(os.environ, GOTOOLCHAIN='local', GOPROXY='off', GOSUMDB='off', CGO_ENABLED='0')
    env.setdefault('GOCACHE', str(work / 'gocache'))
    binary = work / ('csv-quality.exe' if os.name == 'nt' else 'csv-quality')
    subprocess.run(['go', 'build', '-buildvcs=false', '-o', str(binary), '.'], cwd=ROOT, env=env, check=True)
    def run(*args, expected=0):
        result = subprocess.run([str(binary), *map(str,args)], cwd=work, capture_output=True, text=True)
        if result.returncode != expected:
            raise AssertionError((result.returncode, expected, result.stdout, result.stderr))
        return result
    source = work / 'demo.csv'
    run('demo', '-file', source)
    strict = work / 'strict'
    run('check', '-input', source, '-out', strict, expected=3)
    report = json.loads((strict / 'report.json').read_text())
    assert (report['records'], report['accepted'], report['rejected'], report['gate']) == (8,4,4,'fail')
    assert report['totals'] == [{'currency':'EUR','amount_minor':3000}, {'currency':'RUB','amount_minor':39000}, {'currency':'USD','amount_minor':2000}]
    passed = work / 'passed'
    run('check', '-input', source, '-out', passed, '-max-bad-bp', '5000')
    before = (passed / 'report.json').read_bytes()
    run('check', '-input', source, '-out', passed, expected=2)
    assert (passed / 'report.json').read_bytes() == before
    broken = work / 'broken.csv'
    broken.write_bytes(source.read_bytes()+b'"broken\n')
    failed = work / 'failed'
    run('check', '-input', broken, '-out', failed, expected=2)
    assert not failed.exists()
    run('demo', '-file', source, expected=2)
    print('smoke passed: exit 0/2/3, 8 records, 4 accepted, 4 rejected, three currencies, no overwrite, no partial output')
