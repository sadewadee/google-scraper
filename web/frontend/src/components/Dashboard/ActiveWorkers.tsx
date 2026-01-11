import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import type { Worker } from "@/api/types"
import { Circle, Server } from "lucide-react"

interface ActiveWorkersProps {
    workers: Worker[]
}

export function ActiveWorkers({ workers }: ActiveWorkersProps) {
    return (
        <Card className="h-full">
            <CardHeader>
                <CardTitle>Worker Status</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="space-y-8">
                    {workers.map((worker) => (
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
                    {workers.length === 0 && (
                        <div className="text-sm text-muted-foreground text-center py-4">
                            No active workers
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}
