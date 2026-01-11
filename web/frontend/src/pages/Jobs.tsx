import { Button } from "@/components/UI/Button"
import { JobTable } from "@/components/Job/JobTable"
import { Plus } from "lucide-react"
import { Link } from "react-router-dom"

export default function Jobs() {
    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">Jobs Manager</h2>
                <Button asChild>
                    <Link to="/jobs/new">
                        <Plus className="mr-2 h-4 w-4" />
                        New Job
                    </Link>
                </Button>
            </div>

            <JobTable />
        </div>
    )
}
