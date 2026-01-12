import { api } from "./client"
import type { Job, JobCreatePayload, ApiResponse } from "./types"

export const jobsApi = {
    getAll: async (): Promise<ApiResponse<Job[]>> => {
        const response = await api.get<ApiResponse<Job[]>>("/jobs")
        return response.data
    },

    getOne: async (id: string): Promise<ApiResponse<Job>> => {
        const response = await api.get<ApiResponse<Job>>(`/jobs/${id}`)
        return response.data
    },

    create: async (data: JobCreatePayload): Promise<ApiResponse<Job>> => {
        const response = await api.post<ApiResponse<Job>>("/jobs", data)
        return response.data
    },

    cancel: async (id: string): Promise<void> => {
        await api.post(`/jobs/${id}/cancel`)
    },

    delete: async (id: string): Promise<void> => {
        await api.delete(`/jobs/${id}`)
    }
}
