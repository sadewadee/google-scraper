import { api } from "./client"
import type { ResultsResponse } from "./types"

export const resultsApi = {
    // Get all results globally with pagination
    getAll: async (page = 1, perPage = 50): Promise<ResultsResponse> => {
        const response = await api.get<ResultsResponse>("/results", {
            params: { page, per_page: perPage }
        })
        return response.data
    },

    // Get download URL for all results
    downloadUrl: (format: 'csv' | 'json' | 'xlsx', columns?: string[]): string => {
        const baseUrl = api.defaults.baseURL || '/api/v2'
        const params = new URLSearchParams({ format })
        if (columns && columns.length > 0) {
            params.set('columns', columns.join(','))
        }
        return `${baseUrl}/results/download?${params.toString()}`
    }
}
