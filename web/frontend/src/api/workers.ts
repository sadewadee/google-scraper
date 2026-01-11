import { api } from "./client"
import type { Worker, ApiResponse } from "./types"

export const workersApi = {
    getAll: async (): Promise<ApiResponse<Worker[]>> => {
        const response = await api.get<ApiResponse<Worker[]>>("/workers")
        return response.data
    }
}
