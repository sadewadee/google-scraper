import { StatsCards } from "@/components/Dashboard/StatsCards"
import { RecentJobs } from "@/components/Dashboard/RecentJobs"
import { ActiveWorkers } from "@/components/Dashboard/ActiveWorkers"

export default function Dashboard() {
    // Mock data for now
    const stats = {
        total_jobs: 128,
        active_jobs: 12,
        completed_jobs: 110,
        online_workers: 4
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between space-y-2">
                <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
            </div>

            <StatsCards stats={stats} />

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
                <div className="col-span-4">
                    <RecentJobs />
                </div>
                <div className="col-span-3">
                    <ActiveWorkers />
                </div>
            </div>
        </div>
    )
}
