import { useQuery } from "@tanstack/react-query"
import { fetchStats } from "@/api/stats"
import { fetchJobs } from "@/api/jobs"
import { fetchWorkers } from "@/api/workers"

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
        queryFn: () => fetchJobs({ limit: 5 }),
        refetchInterval: 10000,
    })
}

export function useActiveWorkers() {
    return useQuery({
        queryKey: ["active-workers"],
        queryFn: fetchWorkers,
        refetchInterval: 10000,
    })
}
