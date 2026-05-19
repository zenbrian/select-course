// src/api.ts

export interface User {
  id: number;
  username: string;
  created_at: string;
  flag: number;
}

export const login = async (username: string, password: string):Promise<User> => {
  const res = await fetch('/users/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password })
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || '登入失敗');
  }

  return res.json();
};

export const register = async (username: string, password: string):Promise<User> => {
  const res = await fetch('/users/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ username, password })
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || '註冊失敗');
  }

  return res.json();
};

export const logout = async ():Promise<void> => {
  const res = await fetch('/users/logout', {
    method: 'POST',
    credentials: 'include'
  });

  if (!res.ok) {
    throw new Error('登出失敗');
  }
};

export const getUserMe = async ():Promise<User> => {
  const res = await fetch('/users/me', {
    method: 'GET',
    credentials: 'include'
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || '未授權');
  }

  return res.json();
};
