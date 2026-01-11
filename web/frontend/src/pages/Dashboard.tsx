import { StatsCards } from "@/components/Dashboard/StatsCards"
import { RecentJobs } from "@/components/Dashboard/RecentJobs"
import { ActiveWorkers } from "@/components/Dashboard/ActiveWorkers"
import { useDashboardStats, useRecentJobs, useActiveWorkers } from "@/hooks/useDashboard"

export default function Dashboard() {
    const { data: stats, isLoading: isLoadingStats } = useDashboardStats()
    const { data: recentJobs, isLoading: isLoadingJobs } = useRecentJobs()
    const { data: workers, isLoading: isLoadingWorkers } = useActiveWorkers()

    if (isLoadingStats || isLoadingJobs || isLoadingWorkers) {
        return <div>Loading dashboard...</div>
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between space-y-2">
                <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
            </div>

            <StatsCards stats={stats} />

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
                <div className="col-span-4">
                    <RecentJobs jobs={recentJobs?.data || []} />
                </div>
                <div className="col-span-3">
                    <ActiveWorkers workers={workers || []} />
                </div>
            </div>
        </div>
    )
}
