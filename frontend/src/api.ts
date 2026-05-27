export interface User {
  id: number
  username: string
  created_at: string
  flag: number
}

export interface Course {
  id: number
  title: string
  category_id: number
  week:
    | {
        Int32: number
        Valid: boolean
      }
    | number
  duration: string
  capacity: number
  created_at?: string
  updated_at?: string
}

const parseError = async (res: Response, fallback: string): Promise<Error> => {
  const text = await res.text()
  return new Error(text || fallback)
}

export const login = async (username: string, password: string): Promise<User> => {
  const res = await fetch('/users/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  })

  if (!res.ok) {
    throw await parseError(res, '登入失敗')
  }

  return res.json()
}

export const register = async (username: string, password: string): Promise<User> => {
  const res = await fetch('/users/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password }),
  })

  if (!res.ok) {
    throw await parseError(res, '註冊失敗')
  }

  return res.json()
}

export const logout = async (): Promise<void> => {
  const res = await fetch('/users/logout', {
    method: 'POST',
    credentials: 'include',
  })

  if (!res.ok) {
    throw await parseError(res, '登出失敗')
  }
}

export const getUserIndex = async (): Promise<User> => {
  const res = await fetch('/users/index', {
    method: 'GET',
    credentials: 'include',
  })

  if (!res.ok) {
    throw await parseError(res, '尚未登入')
  }

  return res.json()
}

export const getCourses = async (): Promise<Course[]> => {
  const res = await fetch('/courses', {
    method: 'GET',
    credentials: 'include',
  })

  if (!res.ok) {
    throw await parseError(res, '取得課程列表失敗')
  }

  const data = await res.json()
  return data || []
}

export const getUserCourses = async (): Promise<Course[]> => {
  const res = await fetch('/users/index/courses', {
    method: 'GET',
    credentials: 'include',
  })

  if (!res.ok) {
    throw await parseError(res, '取得已選課程失敗')
  }

  const data = await res.json()
  return data || []
}

export const selectCourse = async (courseId: number): Promise<Course> => {
  const res = await fetch('/courses/select', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ course_id: courseId }),
  })

  if (!res.ok) {
    throw await parseError(res, '選課失敗')
  }

  return res.json()
}

export const backCourse = async (courseId: number): Promise<Course> => {
  const res = await fetch('/courses/back-course', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ course_id: courseId }),
  })

  if (!res.ok) {
    throw await parseError(res, '退選失敗')
  }

  return res.json()
}
