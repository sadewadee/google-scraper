import { api } from "./client"
import type { ProxyStats, ProxySource, Proxy, ProxyListResponse, ApiResponse } from "./types"

export interface ProxyListParams {
    page?: number
    limit?: number
    status?: 'pending' | 'healthy' | 'dead' | 'banned'
    country?: string
}

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
    },

    // Proxy list endpoints
    getProxies: async (params: ProxyListParams = {}): Promise<ProxyListResponse> => {
        const queryParams = new URLSearchParams()
        if (params.page) queryParams.set('page', params.page.toString())
        if (params.limit) queryParams.set('limit', params.limit.toString())
        if (params.status) queryParams.set('status', params.status)
        if (params.country) queryParams.set('country', params.country)

        const queryString = queryParams.toString()
        const url = `/proxygate/proxies${queryString ? `?${queryString}` : ''}`
        const response = await api.get<ProxyListResponse>(url)
        return response.data
    },

    cleanupDeadProxies: async (): Promise<ApiResponse<{ message: string; count: number }>> => {
        const response = await api.post<ApiResponse<{ message: string; count: number }>>("/proxygate/proxies/cleanup")
        return response.data
    }
}
