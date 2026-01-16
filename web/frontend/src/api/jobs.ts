import { api } from "./client"
import type { Job, JobCreatePayload, ResultsResponse } from "./types"

export const jobsApi = {
    getAll: async (): Promise<{ data: Job[] }> => {
        const response = await api.get<{ data: Job[] }>("/jobs")
        return response.data
    },

    getOne: async (id: string): Promise<Job> => {
        const response = await api.get<Job>(`/jobs/${id}`)
        return response.data
    },

    create: async (data: JobCreatePayload): Promise<Job> => {
        const response = await api.post<Job>("/jobs", data)
        return response.data
    },

    cancel: async (id: string): Promise<void> => {
        await api.post(`/jobs/${id}/cancel`)
    },

    pause: async (id: string): Promise<void> => {
        await api.post(`/jobs/${id}/pause`)
    },

    resume: async (id: string): Promise<void> => {
        await api.post(`/jobs/${id}/resume`)
    },

    delete: async (id: string): Promise<void> => {
        await api.delete(`/jobs/${id}`)
    },

    getResults: async (id: string, page = 1, perPage = 50): Promise<ResultsResponse> => {
        const response = await api.get<ResultsResponse>(`/jobs/${id}/results`, {
            params: { page, per_page: perPage }
        })
        return response.data
    },

    downloadResults: (id: string, format: 'csv' | 'json' | 'xlsx', columns?: string[]): string => {
        const baseUrl = api.defaults.baseURL || '/api/v2'
        const params = new URLSearchParams({ format })
        if (columns && columns.length > 0) {
            params.set('columns', columns.join(','))
        }
        return `${baseUrl}/jobs/${id}/download?${params.toString()}`
    }
}
