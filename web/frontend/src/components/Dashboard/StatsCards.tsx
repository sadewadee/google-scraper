import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { Activity, CheckCircle, Clock, Users } from "lucide-react"
import { DashboardStats } from "@/api/types"

interface StatsCardsProps {
    stats?: DashboardStats
}

export function StatsCards({ stats }: StatsCardsProps) {
    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Total Jobs</CardTitle>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                    <div className="text-2xl font-bold">{stats?.total_jobs || 0}</div>
                    <p className="text-xs text-muted-foreground">
                        All time
                    </p>
                </CardContent>
            </Card>
            <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Active Jobs</CardTitle>
                    <Clock className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                    <div className="text-2xl font-bold">{stats?.active_jobs || 0}</div>
                    <p className="text-xs text-muted-foreground">
                        Currently processing
                    </p>
                </CardContent>
            </Card>
            <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Completed</CardTitle>
                    <CheckCircle className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                    <div className="text-2xl font-bold">{stats?.completed_jobs || 0}</div>
                    <p className="text-xs text-muted-foreground">
                        Successfully scraped
                    </p>
                </CardContent>
            </Card>
            <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Online Workers</CardTitle>
                    <Users className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                    <div className="text-2xl font-bold">{stats?.online_workers || 0}</div>
                    <p className="text-xs text-muted-foreground">
                        Ready to process
                    </p>
                </CardContent>
            </Card>
        </div>
    )
}
