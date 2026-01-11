import { api } from "./client"
import { DashboardStats } from "./types"

export async function fetchStats(): Promise<DashboardStats> {
    const { data } = await api.get<{ data: DashboardStats }>("/stats")
    return data.data
}
