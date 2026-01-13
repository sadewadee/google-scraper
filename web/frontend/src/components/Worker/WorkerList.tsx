import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { Badge } from "@/components/UI/Badge"
import { Circle, Server, Activity, Clock } from "lucide-react"
import { Link } from "react-router-dom"

// Mock Data
import { useQuery } from "@tanstack/react-query"
import { workersApi } from "@/api/workers"

// Helper function for relative time
function formatRelativeTime(dateStr: string): string {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffSec = Math.floor(diffMs / 1000)
    const diffMin = Math.floor(diffSec / 60)
    const diffHour = Math.floor(diffMin / 60)
    const diffDay = Math.floor(diffHour / 24)

    if (diffSec < 60) return `${diffSec}s ago`
    if (diffMin < 60) return `${diffMin}m ago`
    if (diffHour < 24) return `${diffHour}h ago`
    return `${diffDay}d ago`
}

// Helper function for uptime
function formatUptime(seconds: number): string {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
}

export function WorkerList() {
    const { data: workersRes, isLoading } = useQuery({
        queryKey: ["workers"],
        queryFn: workersApi.getAll,
        refetchInterval: 5000 // Refresh every 5s for worker status
    })

    const workers = workersRes?.data || []

    return (
        <div className="space-y-6">
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Online</CardTitle>
                        <Server className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {isLoading ? "..." : workers.filter(w => w.status !== 'offline').length}
                        </div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Busy Workers</CardTitle>
                        <Activity className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {isLoading ? "..." : workers.filter(w => w.status === 'busy').length}
                        </div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                        <CardTitle className="text-sm font-medium">Total Processed</CardTitle>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {isLoading ? "..." : workers.reduce((acc, curr) => acc + curr.stats.jobs_completed, 0)}
                        </div>
                    </CardContent>
                </Card>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Worker Nodes</CardTitle>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead className="w-[50px]"></TableHead>
                                <TableHead>Worker Name</TableHead>
                                <TableHead>Status</TableHead>
                                <TableHead>Current Job</TableHead>
                                <TableHead className="text-right">Jobs Completed</TableHead>
                                <TableHead className="text-right">Uptime</TableHead>
                                <TableHead className="text-right">Last Heartbeat</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {isLoading ? (
                                <TableRow>
                                    <TableCell colSpan={7} className="h-24 text-center">Loading workers...</TableCell>
                                </TableRow>
                            ) : workers.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={7} className="h-24 text-center">No workers found.</TableCell>
                                </TableRow>
                            ) : (
                                workers.map((worker) => (
                                    <TableRow key={worker.id}>
                                        <TableCell>
                                            <Server className="h-4 w-4 text-muted-foreground" />
                                        </TableCell>
                                        <TableCell className="font-medium">
                                            <div className="font-bold text-foreground">{worker.name}</div>
                                            <div className="text-xs text-muted-foreground font-mono">{worker.id}</div>
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant={worker.status === 'online' ? 'completed' : worker.status === 'busy' ? 'running' : 'failed'}>
                                                <Circle className={`h-2 w-2 mr-1 fill-current`} />
                                                {worker.status}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            {worker.current_job_id ? (
                                                <Link to={`/jobs/${worker.current_job_id}`} className="font-mono text-xs bg-muted px-2 py-1 rounded hover:bg-muted/80">
                                                    #{worker.current_job_id}
                                                </Link>
                                            ) : (
                                                <span className="text-muted-foreground">-</span>
                                            )}
                                        </TableCell>
                                        <TableCell className="text-right">
                                            {worker.stats.jobs_completed.toLocaleString()}
                                        </TableCell>
                                        <TableCell className="text-right text-xs text-muted-foreground">
                                            {formatUptime(worker.stats.uptime_seconds)}
                                        </TableCell>
                                        <TableCell className="text-right text-xs text-muted-foreground">
                                            {formatRelativeTime(worker.last_seen)}
                                        </TableCell>
                                    </TableRow>
                                ))
                            )}
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>
        </div>
    )
}
