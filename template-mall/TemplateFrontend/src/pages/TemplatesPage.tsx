import { useEffect, useMemo, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { api, formatFen, shortEnum } from '../api/client'
import type { Template } from '../api/types'
import { useAuth } from '../auth/AuthContext'

type Category = 'all' | 'word' | 'ppt' | 'excel' | 'pdf'

const CATEGORIES: { id: Category; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'word', label: 'Word' },
  { id: 'ppt', label: 'PPT' },
  { id: 'excel', label: 'Excel' },
  { id: 'pdf', label: 'PDF' },
]

function categoryOf(fileType: string): Category {
  const t = fileType.toLowerCase()
  if (t === 'doc' || t === 'docx') return 'word'
  if (t === 'ppt' || t === 'pptx') return 'ppt'
  if (t === 'xls' || t === 'xlsx') return 'excel'
  if (t === 'pdf') return 'pdf'
  return 'all'
}

function isFree(t: Template): boolean {
  return shortEnum(t.price_type) === 'free' || t.price_fen === 0
}

export function TemplatesPage() {
  const { user, loading } = useAuth()
  const [items, setItems] = useState<Template[]>([])
  const [category, setCategory] = useState<Category>('all')
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')
  const [busyId, setBusyId] = useState('')

  useEffect(() => {
    if (!user) return
    api
      .listTemplates(1, 100)
      .then((res) => setItems(res.items || []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [user])

  const visible = useMemo(() => {
    if (category === 'all') return items
    return items.filter((t) => categoryOf(t.file_type) === category)
  }, [items, category])

  if (!loading && !user) return <Navigate to="/login" replace />

  async function onDownload(t: Template) {
    setMsg('')
    setError('')
    setBusyId(t.template_id)
    try {
      const res = await api.download(t.template_id)
      if (res.result === 'granted') {
        setMsg(`已获得《${t.name}》模版的下载资格，点击“下载”即可完成下载哦！`)
        window.open(res.download_url, '_blank', 'noopener,noreferrer')
        return
      }
      setMsg(`需要支付 ${formatFen(res.payment.amount_fen)}，正在打开支付页…`)
      if (res.payment.pay_url) {
        window.open(res.payment.pay_url, '_blank', 'noopener,noreferrer')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '下载失败')
    } finally {
      setBusyId('')
    }
  }

  return (
    <section>
      <div className="page-head">
        <h1>模板商城</h1>
      </div>

      <div className="category-tabs" role="tablist" aria-label="模板分类">
        {CATEGORIES.map((c) => (
          <button
            key={c.id}
            type="button"
            role="tab"
            aria-selected={category === c.id}
            className={`tab ${category === c.id ? 'active' : ''}`}
            onClick={() => setCategory(c.id)}
          >
            {c.label}
          </button>
        ))}
      </div>

      {error && <p className="error">{error}</p>}
      {msg && <p className="ok">{msg}</p>}

      <div className="template-grid">
        {visible.map((t) => (
          <article key={t.template_id} className="template-card">
            <div className="template-cover">
              {t.cover_url ? (
                <img
                  src={t.cover_url}
                  alt={`${t.name} 首页预览`}
                  loading="lazy"
                  onError={(e) => {
                    const el = e.currentTarget
                    el.style.display = 'none'
                    const fb = el.nextElementSibling as HTMLElement | null
                    if (fb) fb.hidden = false
                  }}
                />
              ) : null}
              <div
                className={`cover-fallback type-${categoryOf(t.file_type)}`}
                hidden={Boolean(t.cover_url)}
              >
                <span>{t.file_type.toUpperCase()}</span>
              </div>
              <span className={`price-tag ${isFree(t) ? 'free' : 'paid'}`}>
                {isFree(t) ? '免费' : formatFen(t.price_fen)}
              </span>
            </div>
            <div className="template-meta">
              <h2 title={t.name}>{t.name}</h2>
              <p className="muted">{categoryOf(t.file_type).toUpperCase()}</p>
              <button
                type="button"
                className="btn primary"
                disabled={busyId === t.template_id}
                onClick={() => void onDownload(t)}
              >
                {busyId === t.template_id ? '处理中…' : '下载'}
              </button>
            </div>
          </article>
        ))}
      </div>

      {visible.length === 0 && <p className="muted empty-hint">该分类暂无已上架模板</p>}
    </section>
  )
}
