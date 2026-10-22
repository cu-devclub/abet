import { useRef, useState, type DragEvent, type ReactNode } from 'react'
import { UploadCloud } from 'lucide-react'
import { cn } from '../../lib/cn'

interface FileDropzoneProps {
  /** Called with picked/dropped plain files. */
  onFiles: (files: File[]) => void
  /** Called synchronously with the raw DataTransfer on drop (needed to read dropped folders). */
  onDrop?: (dt: DataTransfer) => void
  /** When set, a "choose a folder" button is shown and its FileList is passed here. */
  onFolder?: (files: File[]) => void
  accept?: string
  disabled?: boolean
  title: ReactNode
  hint?: ReactNode
  inputLabel: string
}

export function FileDropzone({ onFiles, onDrop, onFolder, accept, disabled, title, hint, inputLabel }: FileDropzoneProps) {
  const [dragging, setDragging] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const folderInput = useRef<HTMLInputElement | null>(null)

  const setFolderInput = (el: HTMLInputElement | null) => {
    folderInput.current = el
    el?.setAttribute('webkitdirectory', '')
  }

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault()
    setDragging(false)
    if (disabled) return
    if (onDrop) onDrop(event.dataTransfer)
    else onFiles(Array.from(event.dataTransfer.files))
  }

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault()
        if (!disabled) setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
      className={cn(
        'border-2 border-dashed rounded-xl px-6 pt-5 pb-6 text-center transition-colors',
        dragging ? 'border-indigo-500 bg-indigo-50/40' : 'border-gray-300 bg-gray-50',
        !disabled && !dragging && 'hover:bg-indigo-50/20 hover:border-indigo-300',
        disabled && 'opacity-60',
      )}
    >
      <UploadCloud className="mx-auto h-12 w-12 text-gray-400" aria-hidden="true" />
      <p className="mt-1 text-sm text-gray-700">{title}</p>
      {hint && <p className="text-xs text-gray-500 mt-1">{hint}</p>}

      <div className="mt-3 flex items-center justify-center gap-2">
        <button
          type="button"
          disabled={disabled}
          onClick={() => fileInput.current?.click()}
          className="px-3 py-1.5 text-xs font-medium text-indigo-700 bg-white border border-indigo-200 rounded-lg hover:bg-indigo-50 transition-colors cursor-pointer disabled:cursor-not-allowed"
        >
          Choose file
        </button>
        {onFolder && (
          <button
            type="button"
            disabled={disabled}
            onClick={() => folderInput.current?.click()}
            className="px-3 py-1.5 text-xs font-medium text-indigo-700 bg-white border border-indigo-200 rounded-lg hover:bg-indigo-50 transition-colors cursor-pointer disabled:cursor-not-allowed"
          >
            Choose folder
          </button>
        )}
      </div>

      <input
        ref={fileInput}
        type="file"
        accept={accept}
        aria-label={inputLabel}
        className="hidden"
        onChange={(e) => {
          onFiles(Array.from(e.target.files ?? []))
          e.target.value = ''
        }}
      />
      {onFolder && (
        <input
          ref={setFolderInput}
          type="file"
          aria-label={`${inputLabel} (folder)`}
          className="hidden"
          multiple
          onChange={(e) => {
            if (e.target.files && e.target.files.length > 0) onFolder(Array.from(e.target.files))
            e.target.value = ''
          }}
        />
      )}
    </div>
  )
}
