import { Button } from "@/components/UI/Button"
import { JobForm } from "@/components/Job/JobForm"
import { ArrowLeft } from "lucide-react"
import { Link, useLocation } from "react-router-dom"
import type { Job } from "@/api/types"

interface LocationState {
    cloneFrom?: Job
    isRetry?: boolean
}

export default function JobCreate() {
    const location = useLocation()
    const state = location.state as LocationState | null
    const cloneFrom = state?.cloneFrom
    const isRetry = state?.isRetry

    return (
        <div className="space-y-6">
            <div className="flex items-center gap-4">
                <Button variant="ghost" size="icon" asChild>
                    <Link to="/jobs">
                        <ArrowLeft className="h-4 w-4" />
                    </Link>
                </Button>
                <h2 className="text-3xl font-bold tracking-tight">
                    {isRetry ? 'Retry Failed Job' : cloneFrom ? 'Clone Job' : 'Create New Scraper Job'}
                </h2>
            </div>

            <div className="max-w-2xl">
                <JobForm cloneFrom={cloneFrom} isRetry={isRetry} />
            </div>
        </div>
    )
}
