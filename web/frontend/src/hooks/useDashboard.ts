import { useQuery } from "@tanstack/react-query"
import { fetchStats } from "@/api/stats"
import { jobsApi } from "@/api/jobs"
import { workersApi } from "@/api/workers"

export function useDashboardStats() {
    return useQuery({
        queryKey: ["dashboard-stats"],
        queryFn: fetchStats,
        refetchInterval: 30000, // Refetch every 30 seconds
    })
}

export function useRecentJobs() {
    return useQuery({
        queryKey: ["recent-jobs"],
        queryFn: () => jobsApi.getAll(),
        refetchInterval: 10000,
    })
}

export function useActiveWorkers() {
    return useQuery({
        queryKey: ["active-workers"],
        queryFn: workersApi.getAll,
        refetchInterval: 10000,
    })
}
