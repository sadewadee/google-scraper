import { api } from "./client"
import type { ProxyStats, ProxySource, ApiResponse } from "./types"

export const proxyApi = {
    getStats: async (): Promise<ApiResponse<ProxyStats>> => {
        const response = await api.get<ApiResponse<ProxyStats>>("/proxygate/stats")
        return response.data
    },

    getSources: async (): Promise<ApiResponse<ProxySource[]>> => {
        const response = await api.get<ApiResponse<ProxySource[]>>("/proxygate/sources")
        return response.data
    },

    addSource: async (url: string): Promise<ApiResponse<ProxySource>> => {
        const response = await api.post<ApiResponse<ProxySource>>("/proxygate/sources", { url })
        return response.data
    },

    deleteSource: async (id: number): Promise<void> => {
        await api.delete(`/proxygate/sources/${id}`)
    },

    refresh: async (): Promise<ApiResponse<{ message: string }>> => {
        const response = await api.post<ApiResponse<{ message: string }>>("/proxygate/refresh")
        return response.data
    },

    updateSource: async (id: number, active: boolean): Promise<ApiResponse<ProxySource>> => {
        const response = await api.patch<ApiResponse<ProxySource>>(`/proxygate/sources/${id}`, { active })
        return response.data
    }
}
