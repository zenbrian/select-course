import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const USER_START_ID = Number(__ENV.USER_START_ID || 1);
const USER_COUNT = Number(__ENV.USER_COUNT || 500);
const TARGET_COURSE_ID = Number(__ENV.TARGET_COURSE_ID || 1);
const THINK_TIME_SECONDS = Number(__ENV.THINK_TIME_SECONDS || 0);

export const options = {
  // One attempt per VU so many users try to select the same course at once.
  scenarios: {
    rush_one_course: {
      executor: 'per-vu-iterations',
      vus: Number(__ENV.VUS || USER_COUNT),
      iterations: Number(__ENV.ITERATIONS_PER_VU || 1),
      maxDuration: __ENV.MAX_DURATION || '30s',
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
    http_req_duration: ['p(95)<800'],
  },
};

export default function () {
  // Use a unique user per VU and force all VUs to compete for one course.
  const userId = USER_START_ID + ((__VU - 1) % USER_COUNT);
  const courseId = TARGET_COURSE_ID;

  const payload = JSON.stringify({
    user_id: userId,
    course_id: courseId,
  });

  const res = http.post(`${BASE_URL}/courses/select`, payload, {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  });

  const acceptableBusinessReject =
    res.status === 400 &&
    (res.body.includes('course already selected') ||
      res.body.includes('time conflict') ||
      res.body.includes('course is full'));

  check(res, {
    'status is success or expected business reject': (r) => r.status === 200 || acceptableBusinessReject,
    'status is not 5xx': (r) => r.status < 500,
  });

  if (THINK_TIME_SECONDS > 0) {
    sleep(THINK_TIME_SECONDS);
  }
}