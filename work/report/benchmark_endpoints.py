import json
import statistics
import time
import urllib.request

profile = ('student.demo@example.test', 'StudyMovie2026!')
targets = [('10k ratings', 8081), ('50k ratings', 8082), ('100k ratings', 8083)]

for label, port in targets:
    base = f'http://127.0.0.1:{port}'
    payload = json.dumps({'email': profile[0], 'password': profile[1]}).encode()
    login = urllib.request.Request(base + '/api/auth/login', data=payload, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(login) as response:
        cookie = response.headers.get('Set-Cookie', '').split(';', 1)[0]
        if not cookie.startswith('session='):
            raise RuntimeError(f'No session cookie from {label}')
    samples = []
    final = None
    for iteration in range(9):
        request = urllib.request.Request(base + '/api/users/me/recommendations', headers={'Cookie': cookie})
        started = time.perf_counter()
        with urllib.request.urlopen(request) as response:
            final = json.load(response)
        elapsed_ms = (time.perf_counter() - started) * 1000
        if iteration > 0:
            samples.append(elapsed_ms)
    ordered = sorted(samples)
    p95 = ordered[min(len(ordered) - 1, int(0.95 * (len(ordered) - 1)))]
    print(json.dumps({
        'dataset': label,
        'samples': len(samples),
        'median_ms': round(statistics.median(samples), 2),
        'mean_ms': round(statistics.mean(samples), 2),
        'p95_ms': round(p95, 2),
        'result_count': len(final['movies']),
        'algorithm': final['algorithm'],
        'similar_users': final.get('similarUsers', 0),
    }))
