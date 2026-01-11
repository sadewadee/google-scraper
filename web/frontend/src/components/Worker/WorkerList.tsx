import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { Circle, Server, Activity, Clock } from "lucide-react"

// Mock Data
import { useQuery } from "@tanstack/react-query"
import { workersApi } from "@/api/workers"

// Mock Data Removed

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
                                <TableHead className="text-right">Last Heartbeat</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {isLoading ? (
                                <TableRow>
                                    <TableCell colSpan={6} className="h-24 text-center">Loading workers...</TableCell>
                                </TableRow>
                            ) : workers.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={6} className="h-24 text-center">No workers found.</TableCell>
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
                                            <div className="flex items-center gap-2">
                                                <Circle className={`h-3 w-3 fill-current ${worker.status === 'online' ? 'text-green-500' :
                                                    worker.status === 'busy' ? 'text-blue-500' :
                                                        'text-gray-400'
                                                    }`} />
                                                <span className="capitalize">{worker.status}</span>
                                            </div>
                                        </TableCell>
                                        <TableCell>
                                            {worker.current_job_id ? (
                                                <span className="font-mono text-xs bg-muted px-2 py-1 rounded">
                                                    #{worker.current_job_id}
                                                </span>
                                            ) : (
                                                <span className="text-muted-foreground">-</span>
                                            )}
                                        </TableCell>
                                        <TableCell className="text-right">
                                            {worker.stats.jobs_completed.toLocaleString()}
                                        </TableCell>
                                        <TableCell className="text-right text-xs text-muted-foreground">
                                            {new Date(worker.last_seen).toLocaleTimeString()}
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
