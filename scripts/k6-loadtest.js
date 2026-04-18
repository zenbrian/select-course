import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// ─── Configuration ──────────────────────────────────────────────────────
const BASE_URL        = __ENV.BASE_URL || 'http://localhost:8081';
const USER_START_ID   = Number(__ENV.USER_START_ID || 1);
const USER_COUNT      = Number(__ENV.USER_COUNT || 500);
const COURSE_START_ID = Number(__ENV.COURSE_START_ID || 1);
const COURSE_COUNT    = Number(__ENV.COURSE_COUNT || 50);
const TARGET_COURSE   = Number(__ENV.TARGET_COURSE_ID || 1);
const SCENARIO        = __ENV.SCENARIO || 'hotspot';

// ─── Custom Metrics ─────────────────────────────────────────────────────
const selectSuccess    = new Counter('select_success');
const selectCourseFull = new Counter('select_course_full');
const selectConflict   = new Counter('select_time_conflict');
const selectDuplicate  = new Counter('select_already_selected');
const selectError      = new Counter('select_server_error');
const selectLatency    = new Trend('select_latency_ms', true);

// ─── Scenario Configurations ────────────────────────────────────────────
//
// Usage:
//   Hotspot:  k6 run --env SCENARIO=hotspot scripts/k6-loadtest.js
//   Spread:   k6 run --env SCENARIO=spread  scripts/k6-loadtest.js
//   Ramp-up:  k6 run --env SCENARIO=rampup  scripts/k6-loadtest.js
//
const SCENARIO_OPTIONS = {
  // 500 users rush 1 course (capacity=20)
  // Measures: lock contention, correct rejection rate
  hotspot: {
    scenarios: {
      hotspot: {
        executor: 'per-vu-iterations',
        vus: Math.min(USER_COUNT, 100),
        iterations: 1,
        maxDuration: '120s',
      },
    },
  },

  // Each user picks a different course from the pool
  // Measures: throughput when lock contention is low
  spread: {
    scenarios: {
      spread: {
        executor: 'per-vu-iterations',
        vus: Math.min(USER_COUNT, 200),
        iterations: 1,
        maxDuration: '120s',
      },
    },
  },

  // Gradually increase concurrent users to find breaking point
  // Measures: p95/p99 under increasing load
  rampup: {
    scenarios: {
      rampup: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
          { duration: '10s', target: 10 },
          { duration: '10s', target: 50 },
          { duration: '10s', target: 100 },
          { duration: '10s', target: 200 },
          { duration: '10s', target: 300 },
          { duration: '10s', target: 0 },
        ],
      },
    },
  },
};

export const options = SCENARIO_OPTIONS[SCENARIO] || SCENARIO_OPTIONS.hotspot;

// ─── Main VU function ───────────────────────────────────────────────────
export default function () {
  const userId = USER_START_ID + ((__VU - 1) % USER_COUNT);

  let courseId;
  if (SCENARIO === 'hotspot') {
    courseId = TARGET_COURSE;
  } else {
    // Pick a random course from the pool
    courseId = COURSE_START_ID + Math.floor(Math.random() * COURSE_COUNT);
  }

  const payload = JSON.stringify({
    user_id: userId,
    course_id: courseId,
  });

  const res = http.post(`${BASE_URL}/courses/select`, payload, {
    headers: { 'Content-Type': 'application/json' },
    timeout: '30s',
  });

  selectLatency.add(res.timings.duration);

  // ── Classify response ──────────────────────────────────────────
  if (res.status === 200) {
    selectSuccess.add(1);
  } else if (res.status === 400) {
    const body = res.body || '';
    if (body.includes('course is full'))          selectCourseFull.add(1);
    else if (body.includes('time conflict'))      selectConflict.add(1);
    else if (body.includes('already selected'))   selectDuplicate.add(1);
  } else if (res.status >= 500) {
    selectError.add(1);
  }

  const acceptableReject =
    res.status === 400 &&
    ((res.body || '').includes('course is full') ||
     (res.body || '').includes('time conflict') ||
     (res.body || '').includes('already selected'));

  check(res, {
    'status is 200 or expected 400': (r) => r.status === 200 || acceptableReject,
    'no server error (5xx)': (r) => r.status < 500,
  });
}

// ─── Summary ────────────────────────────────────────────────────────────
export function handleSummary(data) {
  const m = data.metrics;
  const get = (name) => {
    if (m[name] && m[name].values) return m[name].values.count || 0;
    return 0;
  };

  const lines = [
    '',
    '╔══════════════════════════════════════════════════════════════════╗',
    `║         LOAD TEST RESULTS  —  scenario: ${SCENARIO.padEnd(22)}║`,
    '╚══════════════════════════════════════════════════════════════════╝',
    '',
    '  ── Response Breakdown ───────────────────────────────────────',
    `  ✅ Successful selects     : ${get('select_success')}`,
    `  🚫 Course full (rejected) : ${get('select_course_full')}`,
    `  ⏰ Time conflict          : ${get('select_time_conflict')}`,
    `  🔁 Already selected       : ${get('select_already_selected')}`,
    `  💥 Server errors (5xx)    : ${get('select_server_error')}`,
    '',
  ];

  if (m['http_req_duration'] && m['http_req_duration'].values) {
    const v = m['http_req_duration'].values;
    const fmt = (x) => (x != null ? x.toFixed(1) : 'N/A');
    lines.push('  ── Latency ──────────────────────────────────────────────────');
    lines.push(`  avg  = ${fmt(v.avg)} ms`);
    lines.push(`  med  = ${fmt(v.med)} ms`);
    lines.push(`  p90  = ${fmt(v['p(90)'])} ms`);
    lines.push(`  p95  = ${fmt(v['p(95)'])} ms`);
    lines.push(`  p99  = ${fmt(v['p(99)'])} ms`);
    lines.push(`  max  = ${fmt(v.max)} ms`);
    lines.push('');
  }

  if (m['http_reqs'] && m['http_reqs'].values) {
    lines.push('  ── Throughput ────────────────────────────────────────────────');
    lines.push(`  Total requests : ${m['http_reqs'].values.count}`);
    lines.push(`  Requests/sec   : ${m['http_reqs'].values.rate.toFixed(1)}`);
    lines.push('');
  }

  if (m['http_req_failed'] && m['http_req_failed'].values) {
    const failRate = (m['http_req_failed'].values.rate * 100).toFixed(2);
    lines.push(`  ── Error Rate: ${failRate}% ────────────────────────────────`);
    lines.push('');
  }

  console.log(lines.join('\n'));
  return { stdout: lines.join('\n') };
}
