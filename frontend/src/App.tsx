import { useEffect, useState } from 'react'
import './App.css'
import { getUserMe, login, logout, register, type User } from './api'

function App() {
  const [user, setUser] = useState<User | null>(null)
  const [isLoginView, setIsLoginView] = useState(true)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [checkingSession, setCheckingSession] = useState(true)

  useEffect(() => {
    let cancelled = false

    const bootstrap = async () => {
      try {
        const me = await getUserMe()
        if (!cancelled) {
          setUser(me)
        }
      } catch {
        if (!cancelled) {
          setUser(null)
        }
      } finally {
        if (!cancelled) {
          setCheckingSession(false)
        }
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      if (isLoginView) {
        const data = await login(username, password)
        setUser(data)
      } else {
        await register(username, password)
        const data = await login(username, password)
        setUser(data)
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : '發生未預期錯誤'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = async () => {
    try {
      await logout()
      setUser(null)
      setUsername('')
      setPassword('')
      setError('')
    } catch (err) {
      console.error(err)
    }
  }

  if (checkingSession) {
    return (
      <div className="app-container">
        <div className="auth-card auth-card--loading">
          <h1>登入狀態檢查</h1>
          <p>正在檢查你的登入狀態...</p>
        </div>
      </div>
    )
  }

  if (user) {
    return (
      <div className="app-container">
        <div className="dashboard">
          <div className="dashboard-header">
            <h1>歡迎，{user.username}！</h1>
            <p>你已成功登入。</p>
          </div>
          
          <div className="user-info">
            <div className="info-item">
              <span className="info-label">使用者編號</span>
              <span className="info-value">#{user.id}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Flag</span>
              <span className="info-value">{user.flag}</span>
            </div>
          </div>

          <button className="btn-secondary" onClick={handleLogout}>
            登出
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="app-container">
      <div className="auth-card">
        <div className="auth-header">
          <h1>{isLoginView ? '歡迎回來' : '建立帳號'}</h1>
          <p>{isLoginView ? '登入以使用選課系統' : '加入我們開始學習'}</p>
        </div>

        {error && <div className="error-message">{error}</div>}

        <form className="auth-form" onSubmit={handleSubmit}>
          <div className="input-group">
            <label>使用者名稱</label>
            <input 
              type="text" 
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="請輸入使用者名稱"
              required
            />
          </div>

          <div className="input-group">
            <label>密碼</label>
            <input 
              type="password" 
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </div>

          <button 
            type="submit" 
            className="btn-primary"
            disabled={loading}
          >
            {loading ? '處理中...' : (isLoginView ? '登入' : '註冊')}
          </button>
        </form>

        <div className="auth-toggle">
          {isLoginView ? '還沒有帳號？' : '已經有帳號？'}
          <span onClick={() => {
            setIsLoginView(!isLoginView)
            setError('')
          }}>
            {isLoginView ? '前往註冊' : '前往登入'}
          </span>
        </div>
      </div>
    </div>
  )
}

export default App
