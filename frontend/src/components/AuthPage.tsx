import { useState } from 'react'
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  GraduationCap,
  KeyRound,
  Loader2,
  ShieldCheck,
  UserRound,
} from 'lucide-react'

import { login, register, type User } from '@/api'
import { Button } from '@/components/ui/button'

type AuthTab = 'login' | 'register'

interface AuthPageProps {
  onAuthenticated: (user: User) => void
}

export default function AuthPage({ onAuthenticated }: AuthPageProps) {
  const [tab, setTab] = useState<AuthTab>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const switchTab = (next: AuthTab) => {
    setTab(next)
    setError('')
    setSuccess('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)

    try {
      if (tab === 'login') {
        const user = await login(username, password)
        setSuccess('登入成功，正在進入系統⋯')
        setTimeout(() => onAuthenticated(user), 400)
      } else {
        await register(username, password)
        const user = await login(username, password)
        setSuccess('帳號建立完成！')
        setTimeout(() => onAuthenticated(user), 400)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失敗，請稍後再試')
    } finally {
      setLoading(false)
    }
  }

  const isLogin = tab === 'login'

  return (
    <div
      className="relative flex items-center justify-center overflow-hidden"
      style={{ minHeight: '100vh', padding: '48px 16px' }}
    >
      {/* ── decorative background ── */}
      <div className="pointer-events-none absolute inset-0" style={{ zIndex: 0 }}>
        <div
          className="absolute rounded-full"
          style={{
            top: '-160px',
            left: '50%',
            transform: 'translateX(-50%)',
            width: '520px',
            height: '520px',
            background: 'rgba(0,0,0,0.03)',
            filter: 'blur(100px)',
          }}
        />
        <div
          className="absolute rounded-full"
          style={{
            bottom: '-96px',
            right: '-96px',
            width: '360px',
            height: '360px',
            background: 'rgba(0,0,0,0.02)',
            filter: 'blur(120px)',
          }}
        />
      </div>

      <div style={{ width: '100%', maxWidth: '420px', position: 'relative', zIndex: 1 }}>
        {/* ── brand header ── */}
        <div style={{ marginBottom: '32px', textAlign: 'center' }}>
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '56px',
              height: '56px',
              borderRadius: '16px',
              border: '1px solid var(--line)',
              backgroundColor: 'var(--surface)',
              boxShadow: '0 1px 3px rgba(0,0,0,0.06)',
              marginBottom: '20px',
            }}
          >
            <GraduationCap style={{ width: '28px', height: '28px', color: 'var(--text-primary)' }} />
          </div>
          <h1
            style={{
              fontFamily: 'var(--font-display)',
              fontSize: '2.5rem',
              fontWeight: 700,
              lineHeight: 1,
              letterSpacing: '-0.02em',
              color: 'var(--text-primary)',
            }}
          >
            SELECTCOURSE
          </h1>
          <p
            style={{
              marginTop: '8px',
              fontSize: '14px',
              color: 'var(--text-muted)',
            }}
          >
            智慧選課平台 — 一張表搞定你的學期節奏
          </p>
        </div>

        {/* ── card ── */}
        <div
          style={{
            borderRadius: '16px',
            border: '1px solid var(--line)',
            backgroundColor: 'var(--surface)',
            boxShadow: '0 4px 24px rgba(0,0,0,0.08)',
            overflow: 'hidden',
          }}
        >
          {/* tab bar */}
          <div
            style={{
              display: 'flex',
              borderBottom: '1px solid var(--line)',
            }}
          >
            <button
              type="button"
              onClick={() => switchTab('login')}
              style={{
                flex: 1,
                padding: '14px 0',
                fontSize: '14px',
                fontWeight: 600,
                color: isLogin ? 'var(--text-primary)' : 'var(--text-muted)',
                backgroundColor: 'transparent',
                borderBottom: isLogin ? '2px solid var(--text-primary)' : '2px solid transparent',
                cursor: 'pointer',
                transition: 'color 0.2s, border-color 0.2s',
              }}
            >
              登入
            </button>
            <button
              type="button"
              onClick={() => switchTab('register')}
              style={{
                flex: 1,
                padding: '14px 0',
                fontSize: '14px',
                fontWeight: 600,
                color: !isLogin ? 'var(--text-primary)' : 'var(--text-muted)',
                backgroundColor: 'transparent',
                borderBottom: !isLogin ? '2px solid var(--text-primary)' : '2px solid transparent',
                cursor: 'pointer',
                transition: 'color 0.2s, border-color 0.2s',
              }}
            >
              註冊
            </button>
          </div>

          {/* form body */}
          <div style={{ padding: '24px 28px 28px' }}>
            <div style={{ marginBottom: '20px' }}>
              <h2
                style={{
                  fontSize: '18px',
                  fontWeight: 600,
                  letterSpacing: '-0.01em',
                  color: 'var(--text-primary)',
                }}
              >
                {isLogin ? '歡迎回來' : '建立新帳號'}
              </h2>
              <p
                style={{
                  marginTop: '4px',
                  fontSize: '14px',
                  color: 'var(--text-muted)',
                }}
              >
                {isLogin
                  ? '輸入你的帳號與密碼以繼續。'
                  : '填寫以下欄位即可開始選課。'}
              </p>
            </div>

            {/* status message */}
            {(error || success) && (
              <div
                style={{
                  marginBottom: '20px',
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '10px',
                  padding: '12px 16px',
                  borderRadius: '8px',
                  fontSize: '14px',
                  border: error ? '1px solid rgba(220,38,38,0.3)' : '1px solid rgba(16,185,129,0.3)',
                  backgroundColor: error ? 'rgba(220,38,38,0.05)' : 'rgba(16,185,129,0.05)',
                  color: error ? '#dc2626' : '#059669',
                }}
              >
                {error ? (
                  <AlertCircle style={{ width: '16px', height: '16px', flexShrink: 0, marginTop: '2px' }} />
                ) : (
                  <CheckCircle2 style={{ width: '16px', height: '16px', flexShrink: 0, marginTop: '2px' }} />
                )}
                <span style={{ fontWeight: 500, lineHeight: 1.5 }}>{error || success}</span>
              </div>
            )}

            <form onSubmit={(e) => void handleSubmit(e)}>
              {/* username */}
              <div style={{ marginBottom: '16px' }}>
                <label
                  style={{
                    display: 'block',
                    marginBottom: '6px',
                    fontSize: '12px',
                    fontWeight: 500,
                    color: 'var(--text-muted)',
                  }}
                >
                  使用者帳號
                </label>
                <div style={{ position: 'relative' }}>
                  <UserRound
                    style={{
                      position: 'absolute',
                      left: '12px',
                      top: '50%',
                      transform: 'translateY(-50%)',
                      width: '18px',
                      height: '18px',
                      color: 'var(--text-muted)',
                      opacity: 0.6,
                      pointerEvents: 'none',
                    }}
                  />
                  <input
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="請輸入帳號"
                    required
                    disabled={loading}
                    autoComplete="username"
                    style={{
                      display: 'block',
                      width: '100%',
                      height: '44px',
                      paddingLeft: '42px',
                      paddingRight: '14px',
                      fontSize: '14px',
                      fontFamily: 'var(--font-body)',
                      color: 'var(--text-primary)',
                      backgroundColor: 'var(--surface-soft)',
                      border: '1px solid var(--line)',
                      borderRadius: '10px',
                      outline: 'none',
                      transition: 'border-color 0.2s, box-shadow 0.2s',
                    }}
                    onFocus={(e) => {
                      e.currentTarget.style.borderColor = 'var(--text-muted)'
                      e.currentTarget.style.boxShadow = '0 0 0 3px rgba(0,0,0,0.06)'
                    }}
                    onBlur={(e) => {
                      e.currentTarget.style.borderColor = 'var(--line)'
                      e.currentTarget.style.boxShadow = 'none'
                    }}
                  />
                </div>
              </div>

              {/* password */}
              <div style={{ marginBottom: '20px' }}>
                <label
                  style={{
                    display: 'block',
                    marginBottom: '6px',
                    fontSize: '12px',
                    fontWeight: 500,
                    color: 'var(--text-muted)',
                  }}
                >
                  密碼
                </label>
                <div style={{ position: 'relative' }}>
                  <KeyRound
                    style={{
                      position: 'absolute',
                      left: '12px',
                      top: '50%',
                      transform: 'translateY(-50%)',
                      width: '18px',
                      height: '18px',
                      color: 'var(--text-muted)',
                      opacity: 0.6,
                      pointerEvents: 'none',
                    }}
                  />
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="請輸入密碼"
                    required
                    disabled={loading}
                    autoComplete={isLogin ? 'current-password' : 'new-password'}
                    style={{
                      display: 'block',
                      width: '100%',
                      height: '44px',
                      paddingLeft: '42px',
                      paddingRight: '14px',
                      fontSize: '14px',
                      fontFamily: 'var(--font-body)',
                      color: 'var(--text-primary)',
                      backgroundColor: 'var(--surface-soft)',
                      border: '1px solid var(--line)',
                      borderRadius: '10px',
                      outline: 'none',
                      transition: 'border-color 0.2s, box-shadow 0.2s',
                    }}
                    onFocus={(e) => {
                      e.currentTarget.style.borderColor = 'var(--text-muted)'
                      e.currentTarget.style.boxShadow = '0 0 0 3px rgba(0,0,0,0.06)'
                    }}
                    onBlur={(e) => {
                      e.currentTarget.style.borderColor = 'var(--line)'
                      e.currentTarget.style.boxShadow = 'none'
                    }}
                  />
                </div>
              </div>

              {/* submit */}
              <Button
                type="submit"
                disabled={loading}
                className="h-11 w-full rounded-lg text-sm font-semibold"
              >
                {loading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    處理中⋯
                  </>
                ) : (
                  <>
                    {isLogin ? '登入系統' : '建立帳號'}
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </>
                )}
              </Button>
            </form>

            {/* footer hint */}
            <p
              style={{
                marginTop: '20px',
                textAlign: 'center',
                fontSize: '12px',
                color: 'var(--text-muted)',
              }}
            >
              {isLogin ? (
                <>
                  還沒有帳號？{' '}
                  <button
                    type="button"
                    onClick={() => switchTab('register')}
                    style={{
                      fontWeight: 600,
                      color: 'var(--text-primary)',
                      backgroundColor: 'transparent',
                      textDecoration: 'none',
                      cursor: 'pointer',
                    }}
                    onMouseEnter={(e) => { e.currentTarget.style.textDecoration = 'underline' }}
                    onMouseLeave={(e) => { e.currentTarget.style.textDecoration = 'none' }}
                  >
                    立即註冊
                  </button>
                </>
              ) : (
                <>
                  已有帳號？{' '}
                  <button
                    type="button"
                    onClick={() => switchTab('login')}
                    style={{
                      fontWeight: 600,
                      color: 'var(--text-primary)',
                      backgroundColor: 'transparent',
                      textDecoration: 'none',
                      cursor: 'pointer',
                    }}
                    onMouseEnter={(e) => { e.currentTarget.style.textDecoration = 'underline' }}
                    onMouseLeave={(e) => { e.currentTarget.style.textDecoration = 'none' }}
                  >
                    返回登入
                  </button>
                </>
              )}
            </p>
          </div>
        </div>

        {/* ── bottom badges ── */}
        <div
          style={{
            marginTop: '24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '12px',
            fontSize: '11px',
            color: 'var(--text-muted)',
          }}
        >
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
              padding: '4px 12px',
              borderRadius: '9999px',
              border: '1px solid var(--line)',
              backgroundColor: 'var(--surface)',
            }}
          >
            <ShieldCheck style={{ width: '12px', height: '12px' }} />
            安全連線
          </span>
          <span
            style={{
              padding: '4px 12px',
              borderRadius: '9999px',
              border: '1px solid var(--line)',
              backgroundColor: 'var(--surface)',
            }}
          >
            v2.0.0
          </span>
        </div>
      </div>
    </div>
  )
}
