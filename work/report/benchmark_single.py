import json
import statistics
import sys
import time
import urllib.request

port = int(sys.argv[1])
base = f'http://127.0.0.1:{port}'
data = json.dumps({'email': 'student.demo@example.test', 'password': 'StudyMovie2026!'}).encode()
login = urllib.request.Request(base + '/api/auth/login', data=data, headers={'Content-Type': 'application/json'})
with urllib.request.urlopen(login) as response:
    cookie = response.headers.get('Set-Cookie', '').split(';', 1)[0]
times = []
for i in range(9):
    request = urllib.request.Request(base + '/api/users/me/recommendations', headers={'Cookie': cookie})
    started = time.perf_counter()
    with urllib.request.urlopen(request) as response:
        result = json.load(response)
    elapsed = (time.perf_counter() - started) * 1000
    if i:
        times.append(elapsed)
print(json.dumps({'samples': len(times), 'median_ms': round(statistics.median(times), 2), 'mean_ms': round(statistics.mean(times), 2), 'max_ms': round(max(times), 2), 'results': len(result['movies']), 'neighbors': result.get('similarUsers', 0)}))
