import http from 'k6/http';
import { sleep } from 'k6';

export const options = { vus: 1, iterations: 20 };

export default function () {
  const rps = Math.floor(50 + Math.random()*500);
  const url = __ENV.PREDICT_URL || 'http://localhost:8080/predict';
  const res = http.post(url, JSON.stringify({ rps }), { headers: { 'Content-Type': 'application/json' } });
  console.log(`RPS=${rps} -> ${res.status} ${res.body}`);
  sleep(0.5);
}
