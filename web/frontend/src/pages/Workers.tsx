import { WorkerList } from "@/components/Worker/WorkerList"

export default function Workers() {
    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">Workers Status</h2>
            </div>

            <WorkerList />
        </div>
    )
}
