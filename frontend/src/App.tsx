import { useEffect, useMemo, useState, type ComponentType } from 'react'
import {
  AlertCircle,
  BookOpen,
  Calendar,
  CheckCircle2,
  Clock3,
  Info,
  Loader2,
  LogOut,
  Search,
  ShieldCheck,
  Users,
} from 'lucide-react'

import {
  backCourse,
  getCourses,
  getUserCourses,
  getUserIndex,
  logout,
  selectCourse,
  type Course,
  type User,
} from './api'
import AuthPage from '@/components/AuthPage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

const weekdays = ['週一', '週二', '週三', '週四', '週五']
const sections = [
  { id: '0', name: '上午', time: '09:00 - 12:00' },
  { id: '1', name: '下午', time: '13:30 - 16:30' },
  { id: '2', name: '晚上', time: '18:30 - 21:30' },
]

function App() {
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [checkingSession, setCheckingSession] = useState(true)
  const [courses, setCourses] = useState<Course[]>([])
  const [userCourses, setUserCourses] = useState<Course[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null)
  const [activeTab, setActiveTab] = useState<'catalog' | 'schedule'>('catalog')

  useEffect(() => {
    let cancelled = false

    const bootstrap = async () => {
      try {
        const me = await getUserIndex()
        if (!cancelled) setUser(me)
      } catch {
        if (!cancelled) setUser(null)
      } finally {
        if (!cancelled) setCheckingSession(false)
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (user) void loadCourseData()
  }, [user])

  useEffect(() => {
    if (!error && !success) return

    const timer = window.setTimeout(() => {
      setError('')
      setSuccess('')
    }, 4500)

    return () => window.clearTimeout(timer)
  }, [error, success])

  const loadCourseData = async () => {
    try {
      const [allCourses, selectedCourses] = await Promise.all([getCourses(), getUserCourses()])
      setCourses(allCourses)
      setUserCourses(selectedCourses)
    } catch (err) {
      setError(err instanceof Error ? err.message : '載入課程資料失敗')
    }
  }

  const handleLogout = async () => {
    try {
      await logout()
      setUser(null)
      setCourses([])
      setUserCourses([])
      setError('')
      setSuccess('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '登出失敗')
    }
  }

  const refreshUser = async () => {
    const me = await getUserIndex()
    setUser(me)
  }

  const handleSelectCourse = async (courseId: number) => {
    setError('')
    setSuccess('')
    setActionLoadingId(courseId)

    try {
      const updatedCourse = await selectCourse(courseId)
      setSuccess(`已選取「${updatedCourse.title}」`)
      await loadCourseData()
      await refreshUser()
    } catch (err) {
      setError(err instanceof Error ? err.message : '選課失敗')
    } finally {
      setActionLoadingId(null)
    }
  }

  const handleBackCourse = async (courseId: number) => {
    setError('')
    setSuccess('')
    setActionLoadingId(courseId)

    try {
      const updatedCourse = await backCourse(courseId)
      setSuccess(`已退選「${updatedCourse.title}」`)
      await loadCourseData()
      await refreshUser()
    } catch (err) {
      setError(err instanceof Error ? err.message : '退選失敗')
    } finally {
      setActionLoadingId(null)
    }
  }

  const getWeekVal = (course: Course): number => {
    return typeof course.week === 'object' ? course.week.Int32 : Number(course.week)
  }

  const getWeekText = (course: Course): string => {
    const index = getWeekVal(course)
    return weekdays[index] ?? `週 ${index}`
  }

  const getSection = (course: Course) => {
    return sections.find((section) => section.id === course.duration)
  }

  const getSectionText = (course: Course): string => {
    return getSection(course)?.name ?? `第 ${course.duration} 節`
  }

  const isEnrolled = (courseId: number): boolean => userCourses.some((course) => course.id === courseId)

  const hasTimeConflict = (course: Course): boolean => {
    if (isEnrolled(course.id)) return false

    const courseWeek = getWeekVal(course)
    return userCourses.some((selected) => getWeekVal(selected) === courseWeek && selected.duration === course.duration)
  }

  const timetableMap: Record<string, Course> = {}
  userCourses.forEach((course) => {
    timetableMap[`${getWeekVal(course)}-${course.duration}`] = course
  })

  const categories = useMemo(() => {
    return Array.from(new Set(courses.map((course) => course.category_id))).sort((a, b) => a - b)
  }, [courses])

  const filteredCourses = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase()

    return courses.filter((course) => {
      const matchesSearch =
        normalizedQuery.length === 0 ||
        course.title.toLowerCase().includes(normalizedQuery) ||
        course.id.toString() === normalizedQuery
      const matchesCategory = categoryFilter === 'all' || course.category_id.toString() === categoryFilter

      return matchesSearch && matchesCategory
    })
  }, [categoryFilter, courses, searchQuery])

  const conflictsCount = filteredCourses.filter((course) => hasTimeConflict(course)).length
  const selectedSlots = userCourses.length
  const totalSlots = weekdays.length * sections.length
  const scheduleProgress = totalSlots === 0 ? 0 : (selectedSlots / totalSlots) * 100

  if (checkingSession) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background p-4">
        <Card className="w-full max-w-[400px] border shadow-sm">
          <CardContent className="flex flex-col items-center justify-center gap-4 p-8 text-center">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
            <div className="space-y-1">
              <h1 className="text-xl font-semibold tracking-tight">正在確認登入狀態</h1>
              <p className="text-sm text-muted-foreground">請稍候，系統正在同步你的選課資料。</p>
            </div>
          </CardContent>
        </Card>
      </main>
    )
  }

  if (!user) {
    return <AuthPage onAuthenticated={(u) => setUser(u)} />
  }

  return (
    <main className="mx-auto flex w-full max-w-[1440px] flex-col gap-6">
      <header className="flex flex-col gap-4 border bg-card shadow-sm rounded-xl lg:flex-row lg:items-center lg:justify-between" style={{ padding: '14px 11px 14px 20px' }}>
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <BookOpen className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold tracking-tight">Chi + PG 選課系統</h1>
            <p className="text-sm text-muted-foreground">瀏覽課程、避免衝堂，並即時同步已選課表。</p>
          </div>
        </div>

        <div className="flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between lg:border-l lg:border-t-0 lg:pl-5 lg:pt-0">
          <div className="flex min-w-0 items-center gap-4" style={{ paddingRight: '10px' }}>
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground">使用者</p>
              <p className="max-w-[160px] truncate text-sm font-semibold leading-5">{user.username}</p>
            </div>
            <div className="h-8 w-px bg-border" />
            <div className="shrink-0">
              <p className="text-xs text-muted-foreground">已選</p>
              <p className="text-sm font-semibold leading-5">{userCourses.length} 門課</p>
            </div>
          </div>
          <Button variant="outline" onClick={() => void handleLogout()} className="w-full sm:w-auto rounded-lg">
            <LogOut className="h-4 w-4" />
            登出
          </Button>
        </div>
      </header>

      {(error || success) && (
        <div className="fixed left-1/2 top-4 z-50 w-full max-w-md -translate-x-1/2 px-4">
          <StatusMessage error={error} success={success} floating />
        </div>
      )}

      <section className="grid grid-cols-2 gap-5 md:grid-cols-4">
        <MetricCard icon={BookOpen} label="課程總數" value={`${courses.length}`} />
        <MetricCard icon={ShieldCheck} label="可加入課程" value={`${courses.filter((course) => course.capacity > 0 && !hasTimeConflict(course)).length}`} />
        <MetricCard icon={Calendar} label="已排時段" value={`${selectedSlots}/${totalSlots}`} />
        <MetricCard icon={AlertCircle} label="目前衝堂" value={`${conflictsCount}`} />
      </section>

      <div className="lg:hidden">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'catalog' | 'schedule')}>
          <TabsList className="grid w-full grid-cols-2 rounded-lg">
            <TabsTrigger value="catalog" className="rounded-md">課程清單</TabsTrigger>
            <TabsTrigger value="schedule" className="rounded-md">我的課表</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <section className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_480px]">
        <Card className={cn('shadow-none rounded-xl', activeTab === 'catalog' ? 'block' : 'hidden lg:block')}>
          <CardHeader className="border-b" style={{ padding: '18px 20px' }}>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between" style={{ padding: '2px 0' }}>
              <div className="space-y-1">
                <CardTitle className="text-xl">課程清單</CardTitle>
                <CardDescription>用關鍵字或分類縮小範圍，系統會標示已選、額滿與衝堂狀態。</CardDescription>
              </div>
              <Badge variant="outline">{filteredCourses.length} 筆結果</Badge>
            </div>
          </CardHeader>

          <CardContent className="space-y-4" style={{ padding: '18px 20px 20px' }}>
            <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_180px]" style={{ paddingBottom: '12px' }}>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="搜尋課程名稱或課程編號"
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  className="pl-9 rounded-lg"
                  style={{ paddingLeft: '36px', paddingRight: '12px' }}
                />
              </div>
              <Select value={categoryFilter} onValueChange={setCategoryFilter}>
                <SelectTrigger className="rounded-lg">
                  <SelectValue placeholder="選擇分類" />
                </SelectTrigger>
                <SelectContent className="rounded-lg">
                  <SelectItem value="all">全部分類</SelectItem>
                  {categories.map((categoryId) => (
                    <SelectItem key={categoryId} value={categoryId.toString()}>
                      分類 {categoryId}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid max-h-[calc(100vh-310px)] min-h-[360px] grid-cols-1 gap-4 overflow-y-auto pr-2 xl:grid-cols-2">
              {filteredCourses.length === 0 ? (
                <div className="flex min-h-[260px] items-center justify-center border border-dashed p-8 text-center text-sm text-muted-foreground xl:col-span-2">
                  找不到符合條件的課程。
                </div>
              ) : (
                filteredCourses.map((course) => (
                  <CourseCard
                    key={course.id}
                    course={course}
                    enrolled={isEnrolled(course.id)}
                    conflicted={hasTimeConflict(course)}
                    pending={actionLoadingId === course.id}
                    getWeekText={getWeekText}
                    getSectionText={getSectionText}
                    onSelect={handleSelectCourse}
                    onBack={handleBackCourse}
                  />
                ))
              )}
            </div>
          </CardContent>
        </Card>

        <aside className={cn('space-y-6', activeTab === 'schedule' ? 'block' : 'hidden lg:block')}>
          <Card className="shadow-none rounded-xl">
            <CardHeader className="border-b" style={{ padding: '18px 20px' }}>
              <div className="flex items-center justify-between gap-3" style={{ padding: '2px 0' }}>
                <div>
                  <CardTitle className="text-xl">我的課表</CardTitle>
                  <CardDescription>已選課程會固定在對應時段。</CardDescription>
                </div>
                <Badge variant="secondary">{userCourses.length} 門</Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-5" style={{ padding: '18px 20px 20px' }}>
              <div className="space-y-2.5">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">課表密度</span>
                  <span className="font-medium">{Math.round(scheduleProgress)}%</span>
                </div>
                <Progress value={scheduleProgress} className="h-2" />
              </div>

              <div className="overflow-hidden border bg-background">
                <Table className="table-fixed">
                  <TableHeader className="bg-muted/60">
                    <TableRow>
                      <TableHead className="w-[82px] px-3 py-2 text-center text-xs font-semibold">時段</TableHead>
                      {weekdays.map((day) => (
                        <TableHead key={day} className="px-3 py-2 text-center text-xs font-semibold">
                          {day.replace('週', '')}
                        </TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {sections.map((section) => (
                      <TableRow key={section.id} className="hover:bg-transparent">
                        <TableCell className="h-[92px] border-r bg-muted/30 px-3 py-2 text-center align-middle">
                          <div className="text-xs font-semibold">{section.name}</div>
                          <div className="mt-1 text-[10px] text-muted-foreground">{section.time.split(' - ')[0]}</div>
                        </TableCell>
                        {weekdays.map((_, dayIndex) => {
                          const enrolledCourse = timetableMap[`${dayIndex}-${section.id}`]

                          return (
                            <TableCell key={dayIndex} className="h-[92px] max-h-[92px] overflow-hidden border-r p-2 align-top last:border-r-0">
                              {enrolledCourse ? (
                                <div className="flex h-[76px] min-h-0 flex-col justify-between overflow-hidden border-l-2 border-primary bg-muted/50 px-2.5 py-2 text-left rounded-r-md">
                                  <div className="min-h-0 overflow-hidden">
                                    <p className="line-clamp-2 break-all text-[10px] font-semibold leading-tight text-foreground" title={enrolledCourse.title}>
                                      {enrolledCourse.title}
                                    </p>
                                    <p className="mt-1 truncate text-[10px] leading-none text-muted-foreground">#{enrolledCourse.id}</p>
                                  </div>
                                  <button
                                    type="button"
                                    onClick={() => void handleBackCourse(enrolledCourse.id)}
                                    disabled={actionLoadingId === enrolledCourse.id}
                                    className="mt-2 shrink-0 self-start bg-background px-1.5 py-0.5 text-[10px] font-semibold leading-none text-destructive hover:bg-destructive/10 hover:text-destructive disabled:opacity-60 rounded"
                                  >
                                    {actionLoadingId === enrolledCourse.id ? '處理中' : '退選'}
                                  </button>
                                </div>
                              ) : (
                                <div className="flex h-[76px] items-center justify-center bg-muted/20 text-[10px] text-muted-foreground rounded-md">
                                  空堂
                                </div>
                              )}
                            </TableCell>
                          )
                        })}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>

          <Card className="shadow-none rounded-xl">
            <CardHeader className="border-b" style={{ padding: '5px 5px 5px 20px' }}>
              <CardTitle className="flex items-center gap-2 text-base">
                <Info className="h-4 w-4" />
                選課狀態說明
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1 text-sm text-muted-foreground" style={{ padding: 0 }}>
              <p style={{ padding: '10px 10px 10px 20px' }}>「衝堂」表示該課程與已選課程落在相同星期與時段，系統會停用選課按鈕。容量為 0 時會標示額滿；退選後會重新載入課程與課表資料。</p>
            </CardContent>
          </Card>
        </aside>
      </section>
    </main>
  )
}

interface StatusMessageProps {
  error: string
  success: string
  floating?: boolean
}

function StatusMessage({ error, success, floating = false }: StatusMessageProps) {
  if (!error && !success) return null

  return (
    <div
      className={cn(
        'flex items-center gap-2 border p-3 text-sm shadow-sm',
        floating && 'shadow-lg',
        error ? 'border-destructive/30 bg-destructive text-destructive-foreground' : 'border-emerald-600/20 bg-emerald-600 text-white',
      )}
    >
      {error ? <AlertCircle className="h-4 w-4 shrink-0" /> : <CheckCircle2 className="h-4 w-4 shrink-0" />}
      <span className="font-medium">{error || success}</span>
    </div>
  )
}

interface MetricCardProps {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
}

function MetricCard({ icon: Icon, label, value }: MetricCardProps) {
  return (
    <div className="border bg-card shadow-none rounded-xl" style={{ padding: '16px 18px' }}>
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0" style={{ padding: '1px 0' }}>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className="mt-1 text-2xl font-semibold tracking-tight">{value}</p>
        </div>
        <div className="flex h-9 w-9 items-center justify-center bg-muted/60 rounded-lg">
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
      </div>
    </div>
  )
}

interface CourseCardProps {
  course: Course
  enrolled: boolean
  conflicted: boolean
  pending: boolean
  getWeekText: (course: Course) => string
  getSectionText: (course: Course) => string
  onSelect: (courseId: number) => Promise<void>
  onBack: (courseId: number) => Promise<void>
}

function CourseCard({
  course,
  enrolled,
  conflicted,
  pending,
  getWeekText,
  getSectionText,
  onSelect,
  onBack,
}: CourseCardProps) {
  const full = course.capacity <= 0

  return (
    <Card className={cn('flex flex-col overflow-hidden shadow-none transition-colors hover:bg-muted/20 rounded-xl', enrolled && 'border-primary bg-primary/[0.03]', conflicted && 'opacity-75')}>
      <CardHeader className="space-y-2" style={{ padding: '16px 16px 8px' }}>
        <div className="flex items-start justify-between gap-3" style={{ padding: '1px 0' }}>
          <div className="min-w-0 space-y-1">
            <p className="text-xs text-muted-foreground">課程 #{course.id}</p>
            <CardTitle className="line-clamp-2 text-base leading-snug">{course.title}</CardTitle>
          </div>
          <Badge variant={enrolled ? 'default' : 'secondary'} className="shrink-0">
            分類 {course.category_id}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-3" style={{ padding: '0 16px 12px' }}>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <Calendar className="h-3.5 w-3.5" />
            <span>{getWeekText(course)}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Clock3 className="h-3.5 w-3.5" />
            <span>{getSectionText(course)}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Users className="h-3.5 w-3.5" />
            <span>
              剩餘名額 <strong className="text-foreground">{course.capacity}</strong>
            </span>
          </div>
        </div>

        {full && <p className="text-xs font-medium text-destructive">此課程目前額滿</p>}
      </CardContent>

      <CardFooter style={{ padding: '0 16px 16px' }}>
        {enrolled ? (
          <Button size="sm" variant="outline" className="w-full border-destructive text-destructive hover:bg-destructive/10 hover:text-destructive" disabled={pending} onClick={() => void onBack(course.id)}>
            {pending && <Loader2 className="h-4 w-4 animate-spin" />}
            {pending ? '退選中' : '退選'}
          </Button>
        ) : conflicted ? (
          <Button size="sm" variant="secondary" className="w-full" disabled>
            衝堂
          </Button>
        ) : full ? (
          <Button size="sm" variant="secondary" className="w-full" disabled>
            額滿
          </Button>
        ) : (
          <Button size="sm" className="w-full" disabled={pending} onClick={() => void onSelect(course.id)}>
            {pending && <Loader2 className="h-4 w-4 animate-spin" />}
            {pending ? '選課中' : '加入課表'}
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}

export default App
