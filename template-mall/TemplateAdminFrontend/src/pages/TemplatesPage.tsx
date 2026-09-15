import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
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

function categoryLabel(fileType: string): string {
  const c = categoryOf(fileType)
  if (c === 'all') return fileType.toUpperCase()
  return CATEGORIES.find((x) => x.id === c)?.label || fileType.toUpperCase()
}

function isFree(t: Template): boolean {
  return shortEnum(t.price_type) === 'free' || t.price_fen === 0
}

function isOnShelf(t: Template): boolean {
  return t.status.includes('ON_SHELF')
}

function statusLabel(t: Template): string {
  return isOnShelf(t) ? '上架售卖中' : '已下架'
}

export function TemplatesPage() {
  const { admin, loading } = useAuth()
  const [items, setItems] = useState<Template[]>([])
  const [category, setCategory] = useState<Category>('all')
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')
  const [busyId, setBusyId] = useState('')
  const [editing, setEditing] = useState<Template | null>(null)
  const [editName, setEditName] = useState('')
  const [editPriceType, setEditPriceType] = useState<'free' | 'paid'>('free')
  const [editPriceYuan, setEditPriceYuan] = useState('0')
  const [saving, setSaving] = useState(false)
  const [shelfConfirm, setShelfConfirm] = useState<{
    template: Template
    action: 'publish' | 'unpublish'
  } | null>(null)

  const reload = useCallback(() => {
    return api
      .listTemplates()
      .then((res) => setItems(res.items || []))
      .catch((e) => setError(e instanceof Error ? e.message : '加载失败'))
  }, [])

  useEffect(() => {
    if (!admin) return
    void reload()
  }, [admin, reload])

  const visible = useMemo(() => {
    if (category === 'all') return items
    return items.filter((t) => categoryOf(t.file_type) === category)
  }, [items, category])

  if (!loading && !admin) return <Navigate to="/login" replace />

  async function runShelfAction() {
    if (!shelfConfirm) return
    const { template, action } = shelfConfirm
    setError('')
    setMsg('')
    setBusyId(template.template_id)
    try {
      if (action === 'publish') {
        await api.publish(template.template_id)
        setMsg('上架成功！')
      } else {
        await api.unpublish(template.template_id)
        setMsg('下架成功！')
      }
      setShelfConfirm(null)
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusyId('')
    }
  }

  function openEdit(t: Template) {
    setError('')
    setEditing(t)
    setEditName(t.name)
    const paid = shortEnum(t.price_type) === 'paid' && t.price_fen > 0
    setEditPriceType(paid ? 'paid' : 'free')
    setEditPriceYuan(paid ? (t.price_fen / 100).toFixed(2) : '0')
  }

  function closeEdit() {
    if (saving) return
    setEditing(null)
  }

  async function onSaveEdit(e: FormEvent) {
    e.preventDefault()
    if (!editing) return
    const name = editName.trim()
    if (!name) {
      setError('模板名称不能为空')
      return
    }
    let priceFen = 0
    if (editPriceType === 'paid') {
      const yuan = Number.parseFloat(editPriceYuan || '0')
      if (!Number.isFinite(yuan) || yuan <= 0) {
        setError('付费模板价格必须大于 0')
        return
      }
      priceFen = Math.round(yuan * 100)
    }
    setSaving(true)
    setError('')
    try {
      await api.updateTemplate(editing.template_id, {
        name,
        price_type: editPriceType,
        price_fen: priceFen,
      })
      setEditing(null)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section>
      <div className="page-head">
        <h1>模板管理</h1>
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
        {visible.map((t) => {
          const onShelf = isOnShelf(t)
          return (
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
                <p className="muted">
                  {categoryLabel(t.file_type)} · {statusLabel(t)}
                </p>
                <div className="actions">
                  <button
                    type="button"
                    className="btn ghost"
                    disabled={busyId === t.template_id}
                    onClick={() => openEdit(t)}
                  >
                    编辑
                  </button>
                  {onShelf ? (
                    <button
                      type="button"
                      className="btn danger"
                      disabled={busyId === t.template_id}
                      onClick={() => {
                        setError('')
                        setMsg('')
                        setShelfConfirm({ template: t, action: 'unpublish' })
                      }}
                    >
                      下架
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="btn primary"
                      disabled={busyId === t.template_id}
                      onClick={() => {
                        setError('')
                        setMsg('')
                        setShelfConfirm({ template: t, action: 'publish' })
                      }}
                    >
                      上架
                    </button>
                  )}
                </div>
              </div>
            </article>
          )
        })}
      </div>

      {visible.length === 0 && <p className="muted empty-hint">该分类暂无模板</p>}

      {shelfConfirm && (
        <div
          className="modal-backdrop"
          role="presentation"
          onClick={() => {
            if (!busyId) setShelfConfirm(null)
          }}
        >
          <div className="modal-card" role="dialog" aria-modal="true" onClick={(ev) => ev.stopPropagation()}>
            <h2>{shelfConfirm.action === 'publish' ? '确认上架' : '确认下架'}</h2>
            <p>
              {shelfConfirm.action === 'publish'
                ? `确定将「${shelfConfirm.template.name}」上架售卖吗？`
                : `确定将「${shelfConfirm.template.name}」下架吗？下架后 C 端将不可见。`}
            </p>
            <div className="actions">
              <button
                type="button"
                className="btn ghost"
                disabled={Boolean(busyId)}
                onClick={() => setShelfConfirm(null)}
              >
                取消
              </button>
              <button
                type="button"
                className={shelfConfirm.action === 'publish' ? 'btn primary' : 'btn danger'}
                disabled={Boolean(busyId)}
                onClick={() => void runShelfAction()}
              >
                {busyId
                  ? '处理中…'
                  : shelfConfirm.action === 'publish'
                    ? '确认上架'
                    : '确认下架'}
              </button>
            </div>
          </div>
        </div>
      )}

      {editing && (
        <div className="modal-backdrop" role="presentation" onClick={closeEdit}>
          <form
            className="modal-card"
            onClick={(ev) => ev.stopPropagation()}
            onSubmit={(ev) => void onSaveEdit(ev)}
          >
            <h2>编辑模板</h2>
            <p className="muted mono">{editing.template_id}</p>
            <label>
              模板名称
              <input
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                maxLength={80}
                required
                autoFocus
              />
            </label>
            <label>
              价格类型
              <select
                value={editPriceType}
                onChange={(e) => setEditPriceType(e.target.value as 'free' | 'paid')}
              >
                <option value="free">免费</option>
                <option value="paid">付费</option>
              </select>
            </label>
            {editPriceType === 'paid' && (
              <label>
                零售价（元）
                <input
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={editPriceYuan}
                  onChange={(e) => setEditPriceYuan(e.target.value)}
                  required
                />
              </label>
            )}
            <div className="actions">
              <button type="button" className="btn ghost" disabled={saving} onClick={closeEdit}>
                取消
              </button>
              <button type="submit" className="btn primary" disabled={saving}>
                {saving ? '保存中…' : '保存'}
              </button>
            </div>
          </form>
        </div>
      )}
    </section>
  )
}
