import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MAIN_API } from '../../config/env'
import { apiDelete, uploadWithProgress } from '../../lib/http'
import { downloadAndSave } from '../../lib/download'
import type { Curriculum } from '../curricula/types'

export function coverUrl(curriculumId: string, version?: string): string {
  const query = version ? `?v=${encodeURIComponent(version)}` : ''
  return `${MAIN_API}/curricula/${curriculumId}/cover${query}`
}

function useInvalidateCurricula() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: ['curricula'] })
}

export function useUploadCover() {
  const invalidate = useInvalidateCurricula()
  return useMutation({
    mutationFn: ({ curriculumId, file, onProgress }: { curriculumId: string; file: File; onProgress?: (percent: number) => void }) => {
      const form = new FormData()
      form.append('file', file)
      return uploadWithProgress<Curriculum>(`/curricula/${curriculumId}/cover`, form, onProgress, 'PUT')
    },
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useRemoveCovers() {
  const invalidate = useInvalidateCurricula()
  return useMutation({
    mutationFn: (curriculumIds: string[]) =>
      Promise.all(curriculumIds.map((id) => apiDelete(`/curricula/${id}/cover`))),
    onSuccess: invalidate,
  })
}

export function useDownloadCover() {
  return useMutation({
    mutationFn: (curriculum: Curriculum) =>
      downloadAndSave(`/curricula/${curriculum.id}/cover`, curriculum.cover?.filename ?? 'cover.pdf', {
        params: { download: 1 },
      }),
  })
}
