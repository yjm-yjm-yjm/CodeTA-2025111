import { useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { api, fileExt } from '../api/client'
import { useAuth } from '../auth/AuthContext'

const TEMPLATE_EXTS = ['ppt', 'pptx', 'doc', 'docx', 'xls', 'xlsx', 'pdf']
const COVER_EXTS = ['png', 'jpg', 'jpeg', 'webp']

export function UploadPage() {
  const { admin, loading } = useAuth()
  const [file, setFile] = useState<File | null>(null)
  const [cover, setCover] = useState<File | null>(null)
  const [name, setName] = useState('')
  const [priceYuan, setPriceYuan] = useState('0')
  const [priceType, setPriceType] = useState<'free' | 'paid'>('free')
  const [publish, setPublish] = useState(true)
  const [pending, setPending] = useState(false)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  if (!loading && !admin) return <Navigate to="/login" replace />

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!file) {
      setError('请选择模板文件（ppt/pptx/doc/docx/xls/xlsx/pdf，≤5MB）')
      return
    }
    setError('')
    setMsg('')
    setPending(true)
    try {
      const ext = fileExt(file.name)
      if (!TEMPLATE_EXTS.includes(ext)) {
        throw new Error('仅支持 ppt/pptx/doc/docx/xls/xlsx/pdf')
      }
      if (file.size > 5 * 1024 * 1024) throw new Error('文件不能超过 5MB')

      const contentType = file.type || 'application/octet-stream'
      const cred = await api.uploadCredential(file.name, contentType)
      await api.putFile(cred.upload_url, file, cred.content_type || contentType)

      let coverObjectKey = ''
      if (cover) {
        const cext = fileExt(cover.name)
        if (!COVER_EXTS.includes(cext)) throw new Error('封面仅支持 png/jpg/jpeg/webp')
        const ccred = await api.uploadCredential(cover.name, cover.type || 'image/png')
        await api.putFile(ccred.upload_url, cover, ccred.content_type || cover.type || 'image/png')
        coverObjectKey = ccred.object_key
      }

      const fen =
        priceType === 'free' ? 0 : Math.round(Number.parseFloat(priceYuan || '0') * 100)
      if (priceType === 'paid' && fen <= 0) throw new Error('付费模板价格必须大于 0')

      const templateName = name.trim() || file.name
      const tpl = await api.confirmAndCreate({
        object_key: cred.object_key,
        cover_object_key: coverObjectKey || undefined,
        original_filename: file.name,
        file_type: ext,
        file_size: file.size,
        name: templateName,
        price_fen: fen,
        price_type: priceType,
        publish,
      })
      setMsg(`《${tpl.name || templateName}》模版创建成功！`)
      setFile(null)
      setCover(null)
      setName('')
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setPending(false)
    }
  }

  return (
    <section>
      <div className="page-head">
        <h1>上传并上架模板</h1>
        <p className="muted">
          鉴权直传：模板文件必传；封面可选上传，未传封面时，系统会尝试自动抽取首页图片。
        </p>
      </div>
      <form className="form-card" onSubmit={onSubmit}>
        <label>
          选择模板文件
          <input
            type="file"
            accept=".ppt,.pptx,.doc,.docx,.xls,.xlsx,.pdf"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </label>
        <label>
          首页预览图（可选）
          <input
            type="file"
            accept=".png,.jpg,.jpeg,.webp,image/*"
            onChange={(e) => setCover(e.target.files?.[0] ?? null)}
          />
        </label>
        <label>
          模板名称
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="默认用文件名" />
        </label>
        <label>
          价格类型
          <select
            value={priceType}
            onChange={(e) => setPriceType(e.target.value as 'free' | 'paid')}
          >
            <option value="free">免费</option>
            <option value="paid">付费</option>
          </select>
        </label>
        {priceType === 'paid' && (
          <label>
            零售价（元）
            <input
              type="number"
              min="0.01"
              step="0.01"
              value={priceYuan}
              onChange={(e) => setPriceYuan(e.target.value)}
            />
          </label>
        )}
        <label className="check">
          <input type="checkbox" checked={publish} onChange={(e) => setPublish(e.target.checked)} />
          创建后立即上架
        </label>
        {error && <p className="error">{error}</p>}
        {msg && <p className="ok">{msg}</p>}
        <button className="btn primary" type="submit" disabled={pending}>
          {pending ? '上传中…' : '上传并创建'}
        </button>
      </form>
    </section>
  )
}
