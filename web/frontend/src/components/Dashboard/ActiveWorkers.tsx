import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import type { Worker } from "@/api/types"
import { Circle, Server } from "lucide-react"

// Mock data
const activeWorkers: Worker[] = [
    {
        id: "worker-01",
        name: "Worker 01 - SG",
        status: "online",
        last_seen: "2024-01-20T10:20:00Z",
        stats: { jobs_completed: 150, uptime_seconds: 3600 }
    },
    {
        id: "worker-02",
        name: "Worker 02 - US",
        status: "busy",
        last_seen: "2024-01-20T10:19:55Z",
        current_job_id: 124,
        stats: { jobs_completed: 320, uptime_seconds: 7200 }
    },
    {
        id: "worker-03",
        name: "Worker 03 - DE",
        status: "offline",
        last_seen: "2024-01-19T23:00:00Z",
        stats: { jobs_completed: 45, uptime_seconds: 1200 }
    },
]

export function ActiveWorkers() {
    return (
        <Card className="h-full">
            <CardHeader>
                <CardTitle>Worker Status</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="space-y-8">
                    {activeWorkers.map((worker) => (
                        <div key={worker.id} className="flex items-center">
                            <Server className="h-9 w-9 text-muted-foreground p-2 bg-muted rounded-full mr-4" />
                            <div className="ml-4 space-y-1">
                                <p className="text-sm font-medium leading-none">{worker.name}</p>
                                <p className="text-xs text-muted-foreground">
                                    {worker.status === "busy" ? `Processing Job #${worker.current_job_id}` : `ID: ${worker.id}`}
                                </p>
                            </div>
                            <div className="ml-auto font-medium">
                                <div className="flex items-center gap-2">
                                    <Circle className={`h-3 w-3 fill-current ${worker.status === 'online' ? 'text-green-500' :
                                        worker.status === 'busy' ? 'text-blue-500' :
                                            'text-gray-400'
                                        }`} />
                                    <span className="text-xs capitalize">{worker.status}</span>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    )
}
