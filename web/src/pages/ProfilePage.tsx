import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import Cropper from 'react-easy-crop'
import type { Area } from 'react-easy-crop'
import { Check, Upload, Trash2 } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Avatar } from '@/components/ui/Avatar'
import * as authApi from '@/api/auth'

export function ProfilePage() {
  const { t } = useTranslation()
  const { user, updateUser } = useAuth()

  const [displayName, setDisplayName] = useState(user?.display_name ?? '')
  const [nameSaving, setNameSaving] = useState(false)
  const [nameSaved, setNameSaved] = useState(false)

  const [avatarSaving, setAvatarSaving] = useState(false)
  const [avatarError, setAvatarError] = useState<string | null>(null)
  const [removeConfirmOpen, setRemoveConfirmOpen] = useState(false)

  // Crop state
  const [cropImage, setCropImage] = useState<string | null>(null)
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [croppedArea, setCroppedArea] = useState<Area | null>(null)

  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleSaveName() {
    const trimmed = displayName.trim()
    if (!trimmed || trimmed === user?.display_name) return
    setNameSaving(true)
    try {
      const updated = await authApi.updateProfile(trimmed)
      updateUser(updated)
      setNameSaved(true)
      setTimeout(() => setNameSaved(false), 2000)
    } finally {
      setNameSaving(false)
    }
  }

  function handleFileSelect(file: File) {
    setAvatarError(null)

    if (file.type !== 'image/jpeg' && file.type !== 'image/png') {
      setAvatarError(t('preferences.profile.invalidFileType'))
      return
    }
    if (file.size > 2 * 1024 * 1024) {
      setAvatarError(t('preferences.profile.fileTooLarge'))
      return
    }

    const reader = new FileReader()
    reader.onload = () => setCropImage(reader.result as string)
    reader.readAsDataURL(file)
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) handleFileSelect(file)
    e.target.value = ''
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    const file = e.dataTransfer.files[0]
    if (file) handleFileSelect(file)
  }

  const onCropComplete = useCallback((_: Area, croppedPixels: Area) => {
    setCroppedArea(croppedPixels)
  }, [])

  async function handleCropConfirm() {
    if (!cropImage || !croppedArea) return

    setAvatarSaving(true)
    try {
      const blob = await getCroppedBlob(cropImage, croppedArea)
      const updated = await authApi.uploadAvatar(blob)
      updateUser(updated)
      setCropImage(null)
    } catch {
      setAvatarError('Upload failed')
    } finally {
      setAvatarSaving(false)
    }
  }

  async function handleRemoveAvatar() {
    setAvatarSaving(true)
    try {
      const updated = await authApi.deleteAvatar()
      updateUser(updated)
      setRemoveConfirmOpen(false)
    } finally {
      setAvatarSaving(false)
    }
  }

  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
        {t('preferences.profile.title')}
      </h1>

      <div className="space-y-8">
        {/* Display Name */}
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('preferences.profile.displayName')}
          </h2>
          <div className="flex gap-3 items-center max-w-md">
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t('preferences.profile.displayNamePlaceholder')}
              onKeyDown={(e) => e.key === 'Enter' && handleSaveName()}
            />
            <Button
              onClick={handleSaveName}
              disabled={nameSaving || !displayName.trim() || displayName.trim() === user?.display_name}
            >
              {nameSaved ? <Check className="h-4 w-4 text-green-500" /> : t('preferences.profile.save')}
            </Button>
          </div>
        </div>

        {/* Profile Picture */}
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('preferences.profile.picture')}
          </h2>

          <div className="flex flex-col sm:flex-row items-start gap-6">
            {/* Current avatar preview */}
            <div className="shrink-0">
              <Avatar
                name={user?.display_name ?? ''}
                avatarUrl={user?.avatar_url ? user.avatar_url + (user.avatar_url.includes('?') ? '&' : '?') + 'size=large' : undefined}
                size="xl"
              />
            </div>

            {/* Upload area */}
            <div className="flex-1 max-w-sm">
              <div
                onDrop={handleDrop}
                onDragOver={(e) => e.preventDefault()}
                className="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-4 text-center hover:border-indigo-400 dark:hover:border-indigo-500 transition-colors"
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/png"
                  className="hidden"
                  onChange={handleInputChange}
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={avatarSaving}
                >
                  <Upload className="h-4 w-4 mr-1" />
                  {t('preferences.profile.uploadPicture')}
                </Button>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  {t('preferences.profile.dragDrop')}
                </p>
                <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                  {t('preferences.profile.fileRequirements')}
                </p>
              </div>

              {avatarError && (
                <p className="text-sm text-red-600 dark:text-red-400 mt-2">{avatarError}</p>
              )}

              {user?.avatar_url && (
                <button
                  onClick={() => setRemoveConfirmOpen(true)}
                  className="flex items-center gap-1 text-sm text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 mt-3"
                  disabled={avatarSaving}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  {t('preferences.profile.removePicture')}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Crop Modal */}
      <Modal
        open={!!cropImage}
        onClose={() => setCropImage(null)}
        title={t('preferences.profile.cropTitle')}
      >
        <div className="relative w-full h-64 bg-gray-900">
          {cropImage && (
            <Cropper
              image={cropImage}
              crop={crop}
              zoom={zoom}
              aspect={1}
              cropShape="round"
              onCropChange={setCrop}
              onZoomChange={setZoom}
              onCropComplete={onCropComplete}
            />
          )}
        </div>
        <div className="px-4 py-2">
          <input
            type="range"
            min={1}
            max={3}
            step={0.1}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
            className="w-full"
          />
        </div>
        <div className="flex justify-end gap-3 px-4 pb-4">
          <Button variant="secondary" onClick={() => setCropImage(null)}>
            {t('preferences.profile.cropCancel')}
          </Button>
          <Button onClick={handleCropConfirm} disabled={avatarSaving}>
            {t('preferences.profile.cropConfirm')}
          </Button>
        </div>
      </Modal>

      {/* Remove confirmation */}
      <Modal
        open={removeConfirmOpen}
        onClose={() => setRemoveConfirmOpen(false)}
        title={t('preferences.profile.removePicture')}
      >
        <div className="p-4">
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {t('preferences.profile.removePictureConfirm')}
          </p>
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setRemoveConfirmOpen(false)}>
              {t('preferences.profile.cropCancel')}
            </Button>
            <Button variant="danger" onClick={handleRemoveAvatar} disabled={avatarSaving}>
              {t('preferences.profile.removePicture')}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

// Helper: crop image and return as Blob
async function getCroppedBlob(imageSrc: string, crop: Area): Promise<Blob> {
  const image = await createImage(imageSrc)
  const canvas = document.createElement('canvas')
  canvas.width = crop.width
  canvas.height = crop.height
  const ctx = canvas.getContext('2d')!
  ctx.drawImage(image, crop.x, crop.y, crop.width, crop.height, 0, 0, crop.width, crop.height)
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) resolve(blob)
      else reject(new Error('Canvas to blob failed'))
    }, 'image/jpeg', 0.9)
  })
}

function createImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.addEventListener('load', () => resolve(img))
    img.addEventListener('error', reject)
    img.src = url
  })
}
