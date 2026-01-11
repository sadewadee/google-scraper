import { Button } from "@/components/UI/Button"
import { JobForm } from "@/components/Job/JobForm"
import { ArrowLeft } from "lucide-react"
import { Link } from "react-router-dom"

export default function JobCreate() {
    return (
        <div className="space-y-6">
            <div className="flex items-center gap-4">
                <Button variant="ghost" size="icon" asChild>
                    <Link to="/jobs">
                        <ArrowLeft className="h-4 w-4" />
                    </Link>
                </Button>
                <h2 className="text-3xl font-bold tracking-tight">Create New Scraper Job</h2>
            </div>

            <div className="max-w-2xl">
                <JobForm />
            </div>
        </div>
    )
}
